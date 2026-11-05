"""Build a tiny ONNX classifier and a tokenizer so the Go ONNX Runtime path
can be exercised without any pretrained weights.

The model is deliberately trivial (mean of token embeddings -> linear ->
logits) and the tokenizer is a WordPiece vocabulary trained on the exported
candidate windows. It validates: tokenizer parity between Python and Go, the
ONNX session from Go, dynamic sequence length, and the manifest contract.

    python3 ml/make_test_onnx.py --data data/tierb --out models/tier_b_smoke
"""
import argparse, json, os, random
import numpy as np


def clean_input(s):
    """Mirror of pkg/pii/onnxinput.Clean: the Go runtime applies it to every model input."""
    s = s.replace("[", " ").replace("]", " ").replace("\n", " ⏎ ").replace("\r", "").replace("\t", " ")
    return " ".join(s.split())
import onnx
from onnx import TensorProto, helper, numpy_helper
from tokenizers import Tokenizer, models, normalizers, pre_tokenizers, trainers, processors


def train_tokenizer(texts, vocab_size):
    tok = Tokenizer(models.WordPiece(unk_token="[UNK]"))
    tok.normalizer = normalizers.BertNormalizer(clean_text=True, handle_chinese_chars=False, strip_accents=True, lowercase=True)  # same as the Go runtime
    tok.pre_tokenizer = pre_tokenizers.BertPreTokenizer()
    trainer = trainers.WordPieceTrainer(vocab_size=vocab_size, special_tokens=["[PAD]", "[UNK]", "[CLS]", "[SEP]", "[MASK]"])
    tok.train_from_iterator(texts, trainer)
    cls, sep = tok.token_to_id("[CLS]"), tok.token_to_id("[SEP]")
    tok.post_processor = processors.TemplateProcessing(single="[CLS] $A [SEP]", pair="[CLS] $A [SEP] $B [SEP]", special_tokens=[("[CLS]", cls), ("[SEP]", sep)])
    return tok


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--data", default="data/tierb")
    ap.add_argument("--out", default="models/tier_b_smoke")
    ap.add_argument("--vocab", type=int, default=4000)
    ap.add_argument("--hidden", type=int, default=32)
    args = ap.parse_args()
    os.makedirs(args.out, exist_ok=True)
    random.seed(0)
    texts = []
    with open(os.path.join(args.data, "train_ungated.jsonl")) as f:
        for i, line in enumerate(f):
            if i >= 20000:
                break
            texts.append(json.loads(line)["text"])
    tok = train_tokenizer(texts, args.vocab)
    tok.save(os.path.join(args.out, "tokenizer.json"))

    V, H = tok.get_vocab_size(), args.hidden
    rng = np.random.default_rng(0)
    emb = rng.normal(0, 0.1, (V, H)).astype(np.float32)
    w = rng.normal(0, 0.1, (H, 2)).astype(np.float32)
    b = np.zeros((2,), dtype=np.float32)

    input_ids = helper.make_tensor_value_info("input_ids", TensorProto.INT64, ["batch", "seq"])
    attention_mask = helper.make_tensor_value_info("attention_mask", TensorProto.INT64, ["batch", "seq"])
    logits = helper.make_tensor_value_info("logits", TensorProto.FLOAT, ["batch", 2])
    nodes = [
        helper.make_node("Gather", ["emb", "input_ids"], ["x"], axis=0),                       # [b, s, H]
        helper.make_node("Cast", ["attention_mask"], ["maskf"], to=TensorProto.FLOAT),          # [b, s]
        helper.make_node("Unsqueeze", ["maskf", "axes2"], ["mask3"]),                           # [b, s, 1]
        helper.make_node("Mul", ["x", "mask3"], ["xm"]),
        helper.make_node("ReduceSum", ["xm", "axes1"], ["xs"], keepdims=0),                     # [b, H]
        helper.make_node("ReduceSum", ["maskf", "axes1"], ["cnt"], keepdims=1),                 # [b, 1]
        helper.make_node("Div", ["xs", "cnt"], ["mean"]),
        helper.make_node("MatMul", ["mean", "w"], ["z"]),
        helper.make_node("Add", ["z", "b"], ["logits"]),
    ]
    graph = helper.make_graph(nodes, "smoke", [input_ids, attention_mask], [logits], initializer=[
        numpy_helper.from_array(emb, "emb"), numpy_helper.from_array(w, "w"), numpy_helper.from_array(b, "b"),
        numpy_helper.from_array(np.array([1], dtype=np.int64), "axes1"), numpy_helper.from_array(np.array([2], dtype=np.int64), "axes2"),
    ])
    model = helper.make_model(graph, opset_imports=[helper.make_opsetid("", 17)], producer_name="poc1-smoke")
    model.ir_version = 8
    onnx.checker.check_model(model)
    onnx.save(model, os.path.join(args.out, "model.onnx"))

    # reference outputs for the Go parity test
    import onnxruntime as ort
    sess = ort.InferenceSession(os.path.join(args.out, "model.onnx"), providers=["CPUExecutionProvider"])
    refs = []
    adversarial = [
        "amount | - | Rs ₹ 12,34,567 ⏎ naïve café Ünïcödé « 738192045614 » İstanbul straße ﬁne",
        "name | - | भारत सरकार ⏎ मेरा आधार, मेरी पहचान « Suresh Babu » S/O Venkata",
        "aadhaar | - | [CLS] [SEP] [PAD] [UNK] [MASK] « 738192045614 » [SEP]",
        "pan | aadhaar_no | ẞ Ǆ ǅ « ABCPE1234F » ﬀ ﬁ",
        "",
    ]
    for t in texts[:8] + [clean_input(a) for a in adversarial]:
        enc = tok.encode(t)
        ids = np.array([enc.ids], dtype=np.int64)
        mask = np.array([enc.attention_mask], dtype=np.int64)
        out = sess.run(["logits"], {"input_ids": ids, "attention_mask": mask})[0][0]
        p = 1 / (1 + np.exp(-(out[1] - out[0])))
        refs.append({"text": t, "ids": enc.ids, "p": float(p)})
    json.dump(refs, open(os.path.join(args.out, "reference.json"), "w"), indent=1)
    json.dump({"version": "smoke", "base": "none (random)", "max_len": 128, "inputs": ["input_ids", "attention_mask"],
               "thresholds": {}, "default_threshold": 0.5, "model_file": "model.onnx"}, open(os.path.join(args.out, "manifest.json"), "w"), indent=1)
    print("wrote", args.out, "vocab", V)


if __name__ == "__main__":
    main()
