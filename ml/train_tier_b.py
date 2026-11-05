"""Tier B: a transformer candidate verifier, exported to ONNX for the Go runtime.

Reads the windows written by `go run ./cmd/piiexport` and trains a sequence
classifier on "<type> | <key> | <left> « value » <right>" -> is the claimed
type right. Three modes:

  --base <hf-model>   fine-tune a pretrained encoder (e.g. microsoft/MiniLM-L12-H384-uncased,
                      sentence-transformers/all-MiniLM-L6-v2, microsoft/deberta-v3-xsmall).
                      Needs network access to huggingface.co, or a local path.
  --base scratch      train a small BERT from scratch with a WordPiece tokenizer fitted on the
                      training windows (no network). Useful as a pipeline test and a floor;
                      it cannot generalise like a pretrained model.
  --no-train          only build/export the architecture (random weights) to benchmark latency.

Outputs in --out: tokenizer.json, model.onnx, model.int8.onnx, manifest.json, metrics.json.

    python3 ml/train_tier_b.py --data data/tierb --out models/tier_b --base scratch --arch tiny --epochs 2
"""
import argparse, hashlib, json, math, os, random, sys, time
import numpy as np


def clean_input(s):
    """Mirror of pkg/pii/onnxinput.Clean: the Go runtime applies it to every model input."""
    s = s.replace("[", " ").replace("]", " ").replace("\n", " ⏎ ").replace("\r", "").replace("\t", " ")
    return " ".join(s.split())
import torch
from torch.utils.data import DataLoader, Dataset

ARCHS = {  # layers, hidden, heads, intermediate
    "tiny": (4, 256, 4, 1024),       # ~11M params with an 8k vocab
    "mini": (4, 384, 6, 1536),
    "minilm6": (6, 384, 12, 1536),   # the shape of all-MiniLM-L6-v2 (~22M)
    "small": (6, 512, 8, 2048),
    "base": (12, 768, 12, 3072),     # BERT-base shape (~110M)
}


def read_jsonl(path, limit=None):
    rows = []
    with open(path) as f:
        for i, line in enumerate(f):
            if limit and i >= limit:
                break
            rows.append(json.loads(line))
    return rows


# The Go runtime (github.com/sugarme/tokenizer) implements BertNormalizer with
# explicit settings; "strip_accents": null (what uncased BERT/MiniLM ship) is
# read as false there, and the standalone StripAccents removes different marks.
# Both sides therefore use this exact normalizer, and Go additionally applies
# the same NFD+strip-marks+lowercase before tokenising.
GO_SAFE_NORMALIZER = dict(clean_text=True, handle_chinese_chars=False, strip_accents=True, lowercase=True)

ADVERSARIAL = [
    "aadhaar | - | aadhar « 738192045614 » , bank acc 509347182063",
    "amount | - | Rs ₹ 12,34,567 ⏎ naïve café Ünïcödé « 738192045614 » İstanbul straße ﬁne",
    "name | - | भारत सरकार ⏎ मेरा आधार, मेरी पहचान « Suresh Babu » S/O Venkata",
    "aadhaar | - | [CLS] [SEP] [PAD] [UNK] [MASK] « 738192045614 » [SEP]",
    "mobile | - | " + "word " * 200 + "« 9876543210 » tail",
    "",
    "   ",
    "pan | aadhaar_no | ẞ Ǆ ǅ « ABCPE1234F » ﬀ ﬁ",
]


def fit_tokenizer(texts, vocab_size, out):
    from tokenizers import Tokenizer, models, normalizers, pre_tokenizers, trainers, processors
    tok = Tokenizer(models.WordPiece(unk_token="[UNK]"))
    tok.normalizer = normalizers.BertNormalizer(**GO_SAFE_NORMALIZER)
    tok.pre_tokenizer = pre_tokenizers.BertPreTokenizer()
    tr = trainers.WordPieceTrainer(vocab_size=vocab_size, special_tokens=["[PAD]", "[UNK]", "[CLS]", "[SEP]", "[MASK]"])
    tok.train_from_iterator(texts, tr)
    cls, sep = tok.token_to_id("[CLS]"), tok.token_to_id("[SEP]")
    tok.post_processor = processors.TemplateProcessing(single="[CLS] $A [SEP]", pair="[CLS] $A [SEP] $B [SEP]", special_tokens=[("[CLS]", cls), ("[SEP]", sep)])
    tok.save(os.path.join(out, "tokenizer.json"))
    from transformers import PreTrainedTokenizerFast
    return PreTrainedTokenizerFast(tokenizer_file=os.path.join(out, "tokenizer.json"), unk_token="[UNK]", pad_token="[PAD]", cls_token="[CLS]", sep_token="[SEP]", mask_token="[MASK]")


class Windows(Dataset):
    def __init__(self, rows, tok, max_len):
        self.rows, self.tok, self.max_len = rows, tok, max_len

    def __len__(self):
        return len(self.rows)

    def __getitem__(self, i):
        r = self.rows[i]
        return r["text"], r["label"], r["type"]


def collate(tok, max_len):
    def f(batch):
        texts, labels, types = zip(*batch)
        enc = tok(list(texts), truncation=True, max_length=max_len, padding=True, return_tensors="pt")
        return enc, torch.tensor(labels), types
    return f


def evaluate(model, loader, device):
    model.eval()
    probs, labels, types = [], [], []
    with torch.no_grad():
        for enc, y, t in loader:
            enc = {k: v.to(device) for k, v in enc.items()}
            logits = model(**enc).logits
            p = torch.softmax(logits, -1)[:, 1]
            probs.extend(p.cpu().tolist())
            labels.extend(y.tolist())
            types.extend(t)
    return np.array(probs), np.array(labels), np.array(types)


def _quantile_threshold(s, retain, cap):
    # same rule as eval.ChooseThresholds in Go: k-th smallest TP score with a
    # one-standard-deviation margin so the target holds on unseen data, capped
    n = len(s)
    if n == 0:
        return cap
    margin = math.ceil(math.sqrt(n * retain * (1 - retain)))
    k = max(0, min(n - 1, int(math.floor((1 - retain) * n) - margin)))
    return min(float(np.nextafter(s[k], 0)), cap)


def choose_thresholds(probs, labels, types, retain=0.99, cap=0.5, min_n=100):
    th = {}
    default = _quantile_threshold(np.sort(probs[labels == 1]), retain, cap)
    for t in sorted(set(types)):
        s = np.sort(probs[(labels == 1) & (types == t)])
        th[t] = _quantile_threshold(s, retain, cap) if len(s) >= min_n else default
    return th, default


def ort_session(path, threads=1):
    import onnxruntime as ort
    so = ort.SessionOptions()
    so.intra_op_num_threads = threads
    return ort.InferenceSession(path, so, providers=["CPUExecutionProvider"])


def ort_probs(sess, tok, texts, inputs, max_len, batch=64):
    """P(label=1) from an ONNX session — the int8 model that ships, not the torch one."""
    out = []
    for i in range(0, len(texts), batch):
        enc = tok(list(texts[i:i + batch]), truncation=True, max_length=max_len, padding=True, return_tensors="np")
        feed = {k: enc[k].astype(np.int64) for k in inputs}
        logits = sess.run(["logits"], feed)[0]
        if logits.shape[-1] == 1:
            p = 1 / (1 + np.exp(-logits[:, 0]))
        else:
            z = logits[:, 1] - logits[:, 0]
            p = 1 / (1 + np.exp(-z))
        out.extend(p.tolist())
    return np.array(out)


def file_sha256(path):
    h = hashlib.sha256()
    with open(path, "rb") as f:
        for chunk in iter(lambda: f.read(1 << 20), b""):
            h.update(chunk)
    return h.hexdigest()[:16]


def summarize(probs, labels, types, th, default):
    keep = np.array([p >= th.get(t, default) for p, t in zip(probs, types)])
    tp = int(((labels == 1) & keep).sum())
    fp = int(((labels == 0) & keep).sum())
    tpb, fpb = int((labels == 1).sum()), int((labels == 0).sum())
    return {"tp": tp, "fp": fp, "tp_base": tpb, "fp_base": fpb, "retention": tp / max(tpb, 1), "fp_reduction": 1 - fp / max(fpb, 1), "precision": tp / max(tp + fp, 1)}


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--data", default="data/tierb")
    ap.add_argument("--out", default="models/tier_b")
    ap.add_argument("--base", default="scratch", help="HF model id / local path, or 'scratch'")
    ap.add_argument("--arch", default="tiny", choices=list(ARCHS))
    ap.add_argument("--vocab", type=int, default=8000)
    ap.add_argument("--max-len", type=int, default=128)
    ap.add_argument("--epochs", type=int, default=2)
    ap.add_argument("--batch", type=int, default=32)
    ap.add_argument("--lr", type=float, default=3e-4)
    ap.add_argument("--limit", type=int, default=0, help="cap training rows (0 = all)")
    ap.add_argument("--no-train", action="store_true")
    ap.add_argument("--threads", type=int, default=4)
    args = ap.parse_args()
    torch.set_num_threads(args.threads)
    torch.manual_seed(0)
    random.seed(0)
    os.makedirs(args.out, exist_ok=True)
    device = torch.device("cpu")

    train_rows = read_jsonl(os.path.join(args.data, "train_ungated.jsonl"), args.limit or None)
    dev_rows = read_jsonl(os.path.join(args.data, "dev_gated.jsonl"))
    test_rows = read_jsonl(os.path.join(args.data, "test_gated.jsonl"))
    print(f"train {len(train_rows)} dev {len(dev_rows)} test {len(test_rows)}")

    from transformers import AutoTokenizer, AutoModelForSequenceClassification, BertConfig, BertForSequenceClassification
    if args.base == "scratch":
        tok = fit_tokenizer([r["text"] for r in train_rows], args.vocab, args.out)
        L, H, A, I = ARCHS[args.arch]
        cfg = BertConfig(vocab_size=tok.vocab_size, hidden_size=H, num_hidden_layers=L, num_attention_heads=A, intermediate_size=I,
                         max_position_embeddings=512, num_labels=2, pad_token_id=tok.pad_token_id)
        model = BertForSequenceClassification(cfg)
        base_desc = f"scratch bert {args.arch} L{L} H{H} vocab {tok.vocab_size}"
    else:
        tok = AutoTokenizer.from_pretrained(args.base)
        model = AutoModelForSequenceClassification.from_pretrained(args.base, num_labels=2)
        from tokenizers import normalizers
        bt = tok.backend_tokenizer
        current = json.loads(bt.to_str()).get("normalizer") or {}
        if current.get("type") == "BertNormalizer":
            bt.normalizer = normalizers.BertNormalizer(**GO_SAFE_NORMALIZER)  # before training, so both sides agree
        else:
            print(f"WARNING: tokenizer normalizer is {current.get('type')!r}, not BertNormalizer; the Go runtime lowercases and strips "
                  f"accents before tokenising, so a cased or SentencePiece tokenizer will not match. Use an uncased WordPiece base "
                  f"(MiniLM/BERT-uncased) or verify reference.json carefully.", file=sys.stderr)
        bt.save(os.path.join(args.out, "tokenizer.json"))
        base_desc = args.base
    model.to(device)
    nparams = sum(p.numel() for p in model.parameters())
    print(f"model: {base_desc}, {nparams/1e6:.1f}M parameters")

    metrics = {"base": base_desc, "params": nparams}
    if not args.no_train:
        dl = DataLoader(Windows(train_rows, tok, args.max_len), batch_size=args.batch, shuffle=True, collate_fn=collate(tok, args.max_len))
        opt = torch.optim.AdamW(model.parameters(), lr=args.lr, weight_decay=0.01)
        steps = args.epochs * len(dl)
        sched = torch.optim.lr_scheduler.OneCycleLR(opt, max_lr=args.lr, total_steps=steps, pct_start=0.06)
        t0 = time.time()
        step = 0
        for ep in range(args.epochs):
            model.train()
            for enc, y, _ in dl:
                enc = {k: v.to(device) for k, v in enc.items()}
                out = model(**enc, labels=y.to(device))
                out.loss.backward()
                torch.nn.utils.clip_grad_norm_(model.parameters(), 1.0)
                opt.step(); sched.step(); opt.zero_grad()
                step += 1
                if step % 200 == 0:
                    print(f"epoch {ep} step {step}/{steps} loss {out.loss.item():.4f} {time.time()-t0:.0f}s", flush=True)
        metrics["train_seconds"] = time.time() - t0

    # export; the graph, not the tokenizer, decides which inputs exist (an
    # exporter prunes token_type_ids when the model ignores them)
    model.eval()
    enc = tok(["aadhaar | - | aadhar « 738192045614 » , bank acc 509347182063"], return_tensors="pt")
    tok_inputs = ["input_ids", "attention_mask"] + (["token_type_ids"] if "token_type_ids" in enc else [])
    onnx_path = os.path.join(args.out, "model.onnx")
    torch.onnx.export(model, tuple(enc[k] for k in tok_inputs), onnx_path, input_names=tok_inputs, output_names=["logits"],
                      dynamic_axes={**{k: {0: "batch", 1: "seq"} for k in tok_inputs}, "logits": {0: "batch"}}, opset_version=17, dynamo=False)
    import onnx as onnx_lib
    inputs = [i.name for i in onnx_lib.load(onnx_path).graph.input]
    if inputs != tok_inputs:
        print(f"note: exported graph takes {inputs} (tokenizer produced {tok_inputs}); the manifest records the graph's inputs")
    from onnxruntime.quantization import quantize_dynamic, QuantType
    int8_path = os.path.join(args.out, "model.int8.onnx")
    quantize_dynamic(onnx_path, int8_path, weight_type=QuantType.QInt8)
    metrics["onnx_bytes"] = os.path.getsize(onnx_path)
    metrics["int8_bytes"] = os.path.getsize(int8_path)

    # evaluate and choose thresholds through the int8 ONNX session that ships
    sess8 = ort_session(int8_path, threads=args.threads)
    if not args.no_train:
        p, y, t = ort_probs(sess8, tok, [r["text"] for r in dev_rows], inputs, args.max_len), np.array([r["label"] for r in dev_rows]), np.array([r["type"] for r in dev_rows])
        th, default = choose_thresholds(p, y, t)
        metrics["dev"] = summarize(p, y, t, th, default)
        p, y, t = ort_probs(sess8, tok, [r["text"] for r in test_rows], inputs, args.max_len), np.array([r["label"] for r in test_rows]), np.array([r["type"] for r in test_rows])
        metrics["test"] = summarize(p, y, t, th, default)
        metrics["eval_model"] = "model.int8.onnx via onnxruntime"
        print("dev (int8)", metrics["dev"], "\ntest (int8)", metrics["test"])
    else:
        th, default = {}, 0.5

    # parity references for the Go runtime: real windows plus adversarial strings
    ref_texts = [r["text"] for r in dev_rows[:32]] + [clean_input(a) for a in ADVERSARIAL]
    refs = []
    for text in ref_texts:
        enc_r = tok([text], truncation=True, max_length=args.max_len, return_tensors="np")
        p = float(ort_probs(sess8, tok, [text], inputs, args.max_len)[0])
        refs.append({"text": text, "ids": enc_r["input_ids"][0].tolist(), "p": p})
    json.dump(refs, open(os.path.join(args.out, "reference.json"), "w"), indent=1)

    # python-side latency for reference (Go numbers come from piibench)
    for name, path in [("fp32", onnx_path), ("int8", int8_path)]:
        sess = ort_session(path, threads=1)
        feed = {k: enc[k].numpy() for k in inputs}
        for _ in range(5):
            sess.run(["logits"], feed)
        t0 = time.time(); n = 50
        for _ in range(n):
            sess.run(["logits"], feed)
        metrics[f"py_latency_ms_{name}_1thread"] = (time.time() - t0) / n * 1000
    json.dump({"version": time.strftime("%Y%m%dT%H%M%SZ", time.gmtime()), "base": base_desc, "max_len": args.max_len, "inputs": inputs,
               "thresholds": th, "default_threshold": default, "model_file": "model.int8.onnx", "params": nparams,
               "normalizer": "BertNormalizer" + json.dumps(GO_SAFE_NORMALIZER, sort_keys=True),
               "training_data": f"train_ungated.jsonl sha256:{file_sha256(os.path.join(args.data, 'train_ungated.jsonl'))}",
               "thresholds_from": "model.int8.onnx on dev_gated, retention target 0.99 with margin, cap 0.5"},
              open(os.path.join(args.out, "manifest.json"), "w"), indent=1)
    json.dump(metrics, open(os.path.join(args.out, "metrics.json"), "w"), indent=1)
    print(json.dumps(metrics, indent=1))


if __name__ == "__main__":
    main()
