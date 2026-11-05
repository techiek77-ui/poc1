// Package labels is the catalogue of label phrases ("aadhaar no", "a/c",
// "txn id", ...) that name a value in text, and a tokenizer-based matcher
// that finds them. Every phrase belongs to a Kind; a Kind is either one of the
// PII entity types or a non-PII identifier family (order IDs, transaction
// references, device IDs...) that competes with PII for the same digit
// shapes. Knowing the competing labels is what lets a verifier decide that
// "bank acc 900000000018" is not an Aadhaar number.
package labels

import (
	"sort"
	"strings"

	"github.com/techiek77-ui/poc1/pkg/pii"
)

// Kind is a family of labels. PII kinds use the lowercase entity type name.
type Kind string

const (
	KAadhaar     Kind = "aadhaar"
	KVID         Kind = "vid"
	KPAN         Kind = "pan"
	KGSTIN       Kind = "gstin"
	KCreditCard  Kind = "credit_card"
	KVoterID     Kind = "voter_id"
	KPassport    Kind = "passport"
	KDL          Kind = "driving_licence"
	KIFSC        Kind = "ifsc"
	KMobile      Kind = "mobile"
	KEmail       Kind = "email"
	KUPI         Kind = "upi_id"
	KBankAccount Kind = "bank_account"
	KUAN         Kind = "uan"
	KPincode     Kind = "pincode"
	KDOB         Kind = "dob"
	KName        Kind = "name"
	KAddress     Kind = "address"

	// Non-PII identifier families.
	KOrder     Kind = "order_id"
	KTxn       Kind = "txn_ref"
	KInvoice   Kind = "invoice"
	KTracking  Kind = "tracking"
	KCustomer  Kind = "customer_id"
	KPolicy    Kind = "policy"
	KTicket    Kind = "ticket"
	KPRAN      Kind = "pran"
	KPPO       Kind = "ppo"
	KAPAAR     Kind = "apaar"
	KConsumer  Kind = "consumer_no"
	KDevice    Kind = "device_id"
	KPNR       Kind = "pnr"
	KTimestamp Kind = "timestamp"
	KAmount    Kind = "amount"
	KCheque    Kind = "cheque"
	KVehicle   Kind = "vehicle"
	KEID       Kind = "aadhaar_eid"
	KGeneric   Kind = "generic"
)

// KindType maps a label kind to the PII type it names ("" for non-PII).
var KindType = map[Kind]pii.EntityType{
	KAadhaar: pii.Aadhaar, KVID: pii.VID, KPAN: pii.PAN, KGSTIN: pii.GSTIN,
	KCreditCard: pii.CreditCard, KVoterID: pii.VoterID, KPassport: pii.Passport,
	KDL: pii.DrivingLicence, KIFSC: pii.IFSC, KMobile: pii.Mobile, KEmail: pii.Email,
	KUPI: pii.UPI, KBankAccount: pii.BankAccount, KUAN: pii.UAN, KPincode: pii.Pincode,
	KDOB: pii.DOB, KName: pii.Name, KAddress: pii.Address,
}

// TypeKind is the inverse of KindType.
var TypeKind = func() map[pii.EntityType]Kind {
	m := map[pii.EntityType]Kind{}
	for k, t := range KindType {
		m[t] = k
	}
	return m
}()

// Phrases is the catalogue. Phrases are lowercase; punctuation inside a
// phrase is irrelevant because matching is token based ("a/c" == "a c" ==
// "a.c"). Concatenated identifier forms (accountno, custid) are listed
// explicitly because they are single tokens.
var Phrases = map[Kind][]string{
	KAadhaar: {
		"aadhaar number", "aadhaar no", "aadhaar num", "aadhaar card number", "aadhaar card no",
		"aadhaar card", "aadhaar id", "aadhaar", "aadhar number", "aadhar no", "aadhar card number",
		"aadhar card no", "aadhar card", "aadhar", "adhaar number", "adhaar no", "adhaar", "adhar number",
		"adhar no", "adhar card", "adhar", "aadhaarno", "aadhaarnumber", "aadharno", "aadharnumber",
		"adhaarno", "adharno", "uid number", "uid no", "uidno", "uid", "uidai", "e aadhaar", "eaadhaar",
		"unique identification number", "unique identification", "unique id", "aadhaar seeding",
		"aadhaar linked", "linked aadhaar", "aadhaar details", "aadhaar kyc", "ekyc aadhaar",
		"aadhaar otp", "masked aadhaar", "aadhaar ref", "aadhaar reference", "aadhaar num",
	},
	KVID: {"virtual id", "virtual identification number", "aadhaar vid", "vid number", "vid no", "vid"},
	KPAN: {
		"permanent account number", "permanent account no", "pan card number", "pan card no", "pan number",
		"pan no", "pan num", "pan card", "pan id", "pan details", "pan", "panno", "pannumber", "pancard",
		"income tax pan", "it pan", "pan of", "pan of the",
	},
	KGSTIN: {
		"gst identification number", "gst registration number", "gst registration no", "gst reg no",
		"gstin number", "gstin no", "gstin", "gst number", "gst no", "gst num", "gst id", "gstn", "gst",
		"gstno", "gstnumber", "goods and services tax", "gst of", "gstin of",
	},
	KCreditCard: {
		"credit card number", "credit card no", "credit card", "debit card number", "debit card no",
		"debit card", "card number", "card no", "card num", "cardno", "cardnumber", "card", "visa card",
		"visa", "mastercard", "master card", "rupay card", "rupay", "amex", "american express", "maestro",
		"diners", "atm card", "cc no", "cc number", "cc num", "payment card", "card details", "card ending",
		"cardholder", "card holder",
	},
	KVoterID: {
		"voter id number", "voter id no", "voter id card", "voter id", "voterid", "voter card number",
		"voter card no", "voter card", "voters id", "voter", "epic number", "epic no", "epic", "election card",
		"election id", "electoral id", "electoral roll",
	},
	KPassport: {"passport number", "passport no", "passport num", "passport details", "passport", "passportno", "ppt no", "ppt number"},
	KDL: {
		"driving licence number", "driving licence no", "driving licence", "driving license number",
		"driving license no", "driving license", "driver licence", "driver license", "driving lic",
		"dl number", "dl no", "dlno", "dl", "licence number", "licence no", "license number", "license no",
	},
	KIFSC: {"ifsc code", "ifsc", "ifs code", "ifsccode", "branch code", "branch ifsc", "bank ifsc", "swift code", "swift"},
	KMobile: {
		"mobile number", "mobile no", "mobile num", "mobileno", "mobilenumber", "mobile", "mob no", "mob num",
		"mob", "phone number", "phone no", "phoneno", "phonenumber", "phone", "ph no", "ph", "contact number",
		"contact no", "contactno", "contact", "cell number", "cell no", "cell", "whatsapp number",
		"whatsapp no", "whatsapp", "telephone", "tel", "landline", "msisdn", "registered mobile",
		"registered mobile number", "rmn", "alternate number", "alternate no", "alt no", "alt number",
		"calling number", "caller", "call", "sms to", "otp sent to",
	},
	KEmail: {"email address", "email id", "emailid", "e mail id", "e mail", "email", "mail id", "mailid", "mail"},
	KUPI: {"upi id", "upiid", "upi handle", "upi address", "upi", "vpa", "virtual payment address", "payment address"},
	KBankAccount: {
		"bank account number", "bank account no", "bank account num", "bank account", "bankaccount",
		"bank acc number", "bank acc no", "bank acc", "bank acct no", "bank acct", "bank a c no", "bank a c",
		"bank ac no", "bank ac",
		"account number", "account no", "account num", "accountno", "accountnumber", "acct number", "acct no",
		"acctno", "acct", "acc number", "acc no", "accno", "acc num", "acc", "ac number", "ac no", "acno", "a c",
		"account", "savings account", "savings account number", "savings a c", "sb account", "sb a c", "sb ac",
		"current account", "current a c", "salary account", "salary a c", "beneficiary account",
		"beneficiary a c", "beneficiary account number", "beneficiary acc", "nre account", "nro account",
		"od account", "cc account", "loan account number", "loan account no", "loan account", "loan a c",
		"loan ac no", "khata number", "khata no", "khata", "bank details", "account details", "acc details",
		"a c details", "credited to account", "debited from account", "credited to a c", "debited from a c",
		"from account", "to account", "from a c", "to a c", "account holder", "ac holder",
	},
	KUAN: {
		"universal account number", "universal account no", "uan number", "uan no", "uanno", "uan",
		"epf account number", "epf account no", "epf account", "epf number", "epf no", "epfo number", "epf",
		"epfo", "pf account number", "pf account no", "pf account", "pf number", "pf no", "pfno", "pf a c",
		"pf ac", "pf", "provident fund", "provident fund number", "provident fund account",
	},
	KPincode: {"pin code", "pincode", "postal code", "postcode", "post code", "zip code", "zipcode", "zip", "pin"},
	KDOB: {"date of birth", "dob", "d o b", "birth date", "birthdate", "born on", "born", "dateofbirth"},
	KName: {
		"full name", "customer name", "applicant name", "account holder name", "holder name", "cardholder name",
		"card holder name", "beneficiary name", "nominee name", "employee name", "father name", "father s name",
		"fathers name", "mother name", "mother s name", "spouse name", "husband name", "guardian name",
		"first name", "last name", "middle name", "surname", "given name", "name of applicant",
		"name of customer", "name of account holder", "name as per", "name", "customername", "fullname",
		"firstname", "lastname", "applicant", "beneficiary", "nominee", "proprietor", "proprietor name",
		"authorised signatory", "contact person", "attn", "attention", "dear", "regards", "signed",
		"name of the",
	},
	KAddress: {
		"residential address", "permanent address", "correspondence address", "communication address",
		"present address", "current address", "office address", "billing address", "shipping address",
		"delivery address", "registered address", "address", "addr", "add", "house no", "house number",
		"flat no", "flat number", "plot no", "plot number", "door no", "door number", "h no", "street",
		"locality", "city", "district", "dist", "state", "landmark", "village", "taluk", "taluka", "tehsil",
		"post office", "po", "ps", "near", "opp", "behind", "residing at", "resident of", "located at",
	},
	// --- non-PII identifier families -------------------------------------
	KOrder: {
		"order id", "order no", "order number", "order num", "orderid", "orderno", "order", "ord no", "ord id",
		"ord", "po number", "po no", "purchase order", "purchase order number", "sales order", "so no",
		"so number", "cart id", "basket id", "suborder id", "sub order id", "item id", "sku", "product id",
		"product code", "listing id",
	},
	KTxn: {
		"transaction id", "transaction no", "transaction number", "transaction ref", "transaction reference",
		"transaction reference number", "txn id", "txn no", "txn ref", "txn reference", "txn number", "txnid",
		"txn", "trans id", "tran id", "tranid", "transid", "payment id", "payment ref", "payment reference",
		"payment reference number", "pay id", "payid", "utr number", "utr no", "utr", "rrn number", "rrn no",
		"rrn", "retrieval reference number", "retrieval reference", "imps ref no", "imps reference", "imps ref",
		"imps", "neft ref", "neft reference", "neft utr", "neft", "rtgs utr", "rtgs ref", "rtgs", "upi ref no",
		"upi ref", "upi reference", "upi reference number", "upi txn id", "upi transaction id", "upi txn",
		"ref no", "ref number", "ref num", "refno", "reference no", "reference number", "reference id",
		"ref id", "refid", "reference", "ref", "bank ref", "bank ref no", "bank reference", "bank reference no",
		"arn", "arn no", "auth code", "authorization code", "authorisation code", "approval code", "trace no",
		"trace number", "settlement id", "settlement ref", "payout id", "refund id", "refund ref",
		"transaction", "payment", "remittance ref", "remittance reference", "swift ref", "wire ref",
	},
	KInvoice: {
		"invoice number", "invoice no", "invoice num", "invoice id", "invoiceno", "invoice", "inv no", "inv num",
		"inv id", "inv", "bill number", "bill no", "bill id", "billno", "bill", "receipt number", "receipt no",
		"receipt id", "receipt", "challan number", "challan no", "challan id", "challan", "voucher number",
		"voucher no", "voucher", "debit note", "credit note", "cn no", "dn no", "tax invoice", "e way bill",
		"eway bill", "ewb no", "irn",
	},
	KTracking: {
		"awb number", "awb no", "awb", "tracking id", "tracking number", "tracking no", "tracking code",
		"tracking", "consignment number", "consignment no", "consignment", "shipment id", "shipment number",
		"shipment no", "shipment", "docket number", "docket no", "docket", "lr number", "lr no", "waybill",
		"way bill", "courier ref", "courier reference", "parcel id", "pod no",
	},
	KCustomer: {
		"customer id", "customer number", "customer no", "customer code", "cust id", "custid", "cust no",
		"cust code", "customer ref", "customer reference", "cif number", "cif no", "cif id", "cif", "crn",
		"crn no", "client id", "client code", "client no", "member id", "membership number", "membership no",
		"member no", "employee id", "employee number", "employee no", "employee code", "emp id", "empid",
		"emp code", "emp no", "ecode", "staff id", "staff no", "user id", "userid", "login id", "subscriber id",
		"agent code", "agent id", "dealer code", "dealer id", "merchant id", "merchant code", "mid", "tid",
		"terminal id", "vendor code", "vendor id", "supplier code", "supplier id", "account id", "accountid",
		"profile id", "person id", "party id", "lead id", "contact id", "student id", "roll no", "roll number",
		"enrollment no", "registration number", "registration no", "regn no", "reg no", "application number",
		"application no", "application id", "app id", "appid", "ack no", "acknowledgement number",
		"acknowledgement no", "arn no",
	},
	KPolicy: {
		"policy number", "policy no", "policy num", "policyno", "policy id", "policy", "proposal number",
		"proposal no", "proposal id", "claim number", "claim no", "claim id", "claim", "folio number",
		"folio no", "folio", "certificate no", "certificate number", "cover note", "plan no", "scheme code",
		"lic policy", "insurance policy",
	},
	KTicket: {
		"ticket number", "ticket no", "ticket id", "ticketid", "ticket", "case id", "case number", "case no",
		"case", "incident number", "incident no", "incident id", "incident", "sr number", "sr no", "sr id",
		"service request", "service request number", "service request no", "request id", "requestid",
		"request number", "request no", "req id", "reqid", "req no", "complaint number", "complaint no",
		"complaint id", "complaint", "grievance id", "grievance no", "job id", "jobid", "task id", "taskid",
		"session id", "sessionid", "trace id", "traceid", "span id", "spanid", "correlation id",
		"correlationid", "message id", "msg id", "msgid", "batch id", "batchid", "batch no", "run id", "runid",
		"event id", "eventid", "log id", "record id", "recordid", "row id", "rowid", "seq no", "sequence no",
		"sequence number", "sl no", "s no", "sr", "serial", "process id", "pid", "thread id", "build no",
		"build number", "version",
	},
	KPRAN:  {"pran number", "pran no", "pran", "nps account", "nps account number", "nps"},
	KPPO:   {"ppo number", "ppo no", "ppo", "pension number", "pension no", "pension id", "pension payment order"},
	KAPAAR: {"apaar id", "apaar number", "apaar", "abc id", "academic bank of credits"},
	KConsumer: {
		"consumer number", "consumer no", "consumer id", "consumer code", "consumerno", "ca number", "ca no",
		"k number", "k no", "service number", "service no", "connection number", "connection no", "connection id",
		"meter number", "meter no", "meter id", "lpg id", "lpg consumer no", "lpg consumer number",
		"bp number", "bp no", "rr number", "rr no", "ivrs number", "ivrs no", "subscriber number",
		"subscriber no", "consumer account", "electricity account", "gas connection", "water connection",
		"account no electricity", "bill account no",
	},
	KDevice: {
		"imei number", "imei no", "imei", "device id", "deviceid", "device", "serial number", "serial no",
		"serialno", "sn", "s n", "mac address", "mac", "esn", "meid", "sim number", "sim no", "sim", "iccid",
		"imsi", "model no", "model number", "asset id", "asset tag", "asset no", "hostname", "ip address", "ip",
	},
	KPNR: {
		"pnr number", "pnr no", "pnr", "booking id", "booking ref", "booking reference", "booking number",
		"booking no", "reservation number", "reservation no", "reservation id", "confirmation number",
		"confirmation no", "confirmation code", "ticket pnr", "flight booking", "train ticket",
	},
	KTimestamp: {
		"timestamp", "time stamp", "ts", "epoch", "epoch ms", "epoch time", "unix time", "created at",
		"createdat", "updated at", "updatedat", "event time", "eventtime", "logged at", "received at",
		"sent at", "start time", "end time", "expiry", "expires at", "valid till", "generated at",
	},
	KAmount: {
		"amount", "amt", "rs", "inr", "rupees", "balance", "total", "total amount", "net amount",
		"gross amount", "price", "fee", "charges", "charge", "credit", "debit", "payable", "paid", "due",
		"outstanding", "emi", "premium", "principal", "interest", "salary", "ctc", "mrp", "discount", "tax",
		"gst amount", "cgst", "sgst", "igst", "tds", "value",
	},
	KCheque: {
		"cheque number", "cheque no", "cheque num", "cheque", "chq no", "chq", "check no", "check number",
		"dd number", "dd no", "demand draft", "micr code", "micr", "cheque leaf",
	},
	KVehicle: {
		"vehicle number", "vehicle no", "vehicle registration", "vehicle reg no", "registration mark", "rc number",
		"rc no", "rc", "chassis number", "chassis no", "engine number", "engine no", "vin", "fastag id", "fastag",
		"vehicle", "reg number",
	},
	KEID: {
		"enrolment number", "enrolment no", "enrolment id", "enrollment number", "enrollment id", "eid",
		"aadhaar enrolment number", "aadhaar enrolment no", "aadhaar enrollment no", "enrolment slip",
	},
	KGeneric: {
		"number", "no", "num", "id", "identifier", "code", "key", "value", "data", "info", "details", "field",
		"input", "entry", "numeric", "digits",
	},
}

// Token is a lowercase alphanumeric run with its byte offsets in the source.
type Token struct {
	Text  string
	Norm  string // OCR-normalised variant (0->o, 1->l, 5->s, 8->b) when the token is mostly letters; else == Text
	Start int
	End   int
}

// Tokenize splits text into lowercase [a-z0-9]+ tokens, breaking on
// underscores and camelCase boundaries so that "aadhaarNo", "aadhaar_no" and
// "Aadhaar No" tokenize identically. Offsets are byte offsets in text.
func Tokenize(text string) []Token {
	var toks []Token
	start := -1
	prevLower := false
	flush := func(end int) {
		if start >= 0 && end > start {
			raw := text[start:end]
			toks = append(toks, Token{Text: strings.ToLower(raw), Norm: ocrNorm(strings.ToLower(raw)), Start: start, End: end})
		}
		start = -1
	}
	for i := 0; i < len(text); i++ {
		c := text[i]
		isUpper := c >= 'A' && c <= 'Z'
		isLower := c >= 'a' && c <= 'z'
		isDigit := c >= '0' && c <= '9'
		switch {
		case isUpper:
			if start >= 0 && prevLower { // camelCase boundary
				flush(i)
			}
			if start < 0 {
				start = i
			}
		case isLower || isDigit:
			if start < 0 {
				start = i
			}
		default:
			flush(i)
		}
		prevLower = isLower
	}
	flush(len(text))
	return toks
}

// ocrNorm maps the classic OCR digit/letter confusions back to letters when
// a token is mostly alphabetic ("acc0unt" -> "account", "m0bile" -> "mobile").
func ocrNorm(t string) string {
	letters, digits := 0, 0
	for i := 0; i < len(t); i++ {
		switch {
		case t[i] >= 'a' && t[i] <= 'z':
			letters++
		case t[i] >= '0' && t[i] <= '9':
			digits++
		}
	}
	if digits == 0 || letters == 0 || letters < digits {
		return t
	}
	r := strings.NewReplacer("0", "o", "1", "l", "5", "s", "8", "b", "2", "z", "6", "b", "7", "t")
	return r.Replace(t)
}

// Weak phrases are genuinely ambiguous on their own ("uid" is a user id or
// an Aadhaar, "card" is any card, "pin" an ATM PIN or a postal code). They
// still gate and still count, but a verifier should not treat them as strong
// evidence for or against a claim.
var Weak = map[string]bool{
	"uid": true, "card": true, "pin": true, "zip": true, "ref": true, "reference": true, "cc": true, "ac": true,
	"acc": true, "sn": true, "mac": true, "ip": true, "call": true, "mail": true, "mob": true, "ph": true, "tel": true,
	"payment": true, "transaction": true, "credit": true, "debit": true, "serial": true, "device": true, "time": true,
	"ts": true, "dear": true, "regards": true, "attn": true, "attention": true, "signed": true, "beneficiary": true,
	"nominee": true, "applicant": true, "proprietor": true, "city": true, "state": true, "district": true,
	"street": true, "post": true, "po": true, "ps": true, "near": true, "opp": true, "behind": true, "add": true,
	"addr": true, "pf": true, "case": true, "vehicle": true, "user": true, "order": true, "bill": true, "receipt": true,
	"voucher": true, "challan": true, "policy": true, "claim": true, "ticket": true, "incident": true, "folio": true,
	"shipment": true, "tracking": true, "consignment": true, "docket": true, "sim": true, "model no": true, "sr": true,
	"swift": true, "landline": true, "epf": true, "epfo": true, "nps": true, "pension": true, "value": true, "data": true,
	"gst": true, "invoice": true, "cheque": true, "voter": true, "passport": false, "name": false,
}

// Hit is a matched label phrase.
type Hit struct {
	Kind   Kind
	Type   pii.EntityType
	Phrase string
	Start  int // byte offsets in the source text
	End    int
	NTok   int
	OCR    bool // matched only after OCR normalisation
	Weak   bool // see Weak
}

type phraseKey struct {
	kind   Kind
	phrase string
}

var (
	// index: first token -> candidate phrases (as token slices), longest first
	index   map[string][]phraseEntry
	maxTok  int
	builtOK bool
)

type phraseEntry struct {
	toks   []string
	kind   Kind
	phrase string
}

func build() {
	if builtOK {
		return
	}
	index = map[string][]phraseEntry{}
	for kind, phrases := range Phrases {
		for _, p := range phrases {
			toks := strings.Fields(p)
			if len(toks) == 0 {
				continue
			}
			if len(toks) > maxTok {
				maxTok = len(toks)
			}
			index[toks[0]] = append(index[toks[0]], phraseEntry{toks: toks, kind: kind, phrase: p})
		}
	}
	for k := range index {
		sort.SliceStable(index[k], func(i, j int) bool { return len(index[k][i].toks) > len(index[k][j].toks) })
	}
	builtOK = true
}

// Find returns every label phrase occurrence in text. At each token position
// the longest matching phrase wins and shorter phrases starting at the same
// token are not reported; overlapping phrases starting at later tokens are
// reported (so "permanent account number" also yields "account number" one
// token later only if the longest-match consumed fewer tokens, which it
// doesn't). Hits are sorted by Start.
func Find(text string) []Hit {
	build()
	toks := Tokenize(text)
	return FindTokens(toks)
}

// FindTokens is Find over pre-computed tokens.
func FindTokens(toks []Token) []Hit {
	build()
	var hits []Hit
	consumedUntil := -1 // token index up to which a longer phrase already consumed
	for i := range toks {
		best := matchAt(toks, i, false)
		ocr := false
		if best == nil {
			best = matchAt(toks, i, true)
			ocr = best != nil
		}
		if best == nil {
			continue
		}
		n := len(best.toks)
		if i < consumedUntil {
			// A longer phrase starting earlier already covers this token.
			// Still report if this phrase extends beyond the earlier one.
			if i+n <= consumedUntil {
				continue
			}
		}
		hits = append(hits, Hit{
			Kind: best.kind, Type: KindType[best.kind], Phrase: best.phrase,
			Start: toks[i].Start, End: toks[i+n-1].End, NTok: n, OCR: ocr, Weak: Weak[best.phrase],
		})
		if i+n > consumedUntil {
			consumedUntil = i + n
		}
	}
	return hits
}

func matchAt(toks []Token, i int, useNorm bool) *phraseEntry {
	first := toks[i].Text
	if useNorm {
		first = toks[i].Norm
	}
	cands, ok := index[first]
	if !ok {
		return nil
	}
	for ci := range cands {
		pe := &cands[ci]
		if i+len(pe.toks) > len(toks) {
			continue
		}
		match := true
		for j := 1; j < len(pe.toks); j++ {
			t := toks[i+j].Text
			if useNorm {
				t = toks[i+j].Norm
			}
			if t != pe.toks[j] {
				match = false
				break
			}
		}
		if match {
			return pe
		}
	}
	return nil
}

// IsPII reports whether the kind names a PII entity type.
func (k Kind) IsPII() bool { return KindType[k] != "" }

// KindsSorted returns all kinds in a stable order.
func KindsSorted() []Kind {
	ks := make([]Kind, 0, len(Phrases))
	for k := range Phrases {
		ks = append(ks, k)
	}
	sort.Slice(ks, func(i, j int) bool { return ks[i] < ks[j] })
	return ks
}
