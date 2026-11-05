package synth

import (
	"fmt"
	"hash/fnv"
	"math/rand"
	"strings"
	"time"

	"github.com/techiek77-ui/poc1/pkg/pii"
	"github.com/techiek77-ui/poc1/pkg/pii/checksum"
	"github.com/techiek77-ui/poc1/pkg/pii/labels"
)

// Field is one kind of value that can appear in a document.
type Field struct {
	Kind   string         // matches labels.Kind where one exists
	Type   pii.EntityType // "" for non-PII
	Labels []string       // label variants; pooled into train-only / test-only / shared by hash
	Gen    func(r *rand.Rand, p *Person) string
	Weight int // relative frequency when sampling distractors / extra fields
}

// Pool returns "train", "test" or "" (shared) for a label or template string.
// It is a deterministic hash so the split is reproducible and so that the
// held-out test set contains phrasings never seen in training.
func Pool(s string) string {
	h := fnv.New32a()
	h.Write([]byte(s))
	switch h.Sum32() % 10 {
	case 0, 1:
		return "test"
	case 2, 3:
		return "train"
	default:
		return ""
	}
}

// allowed reports whether a pooled string may be used for a split.
func allowed(s, split string) bool {
	p := Pool(s)
	if p == "" {
		return true
	}
	if split == "train" {
		return p == "train"
	}
	return p == "test"
}

// pickPooled chooses one of xs allowed for split.
func pickPooled(r *rand.Rand, xs []string, split string) string {
	for tries := 0; tries < 20; tries++ {
		s := xs[r.Intn(len(xs))]
		if allowed(s, split) {
			return s
		}
	}
	for _, s := range xs { // fall back to the first allowed
		if allowed(s, split) {
			return s
		}
	}
	return xs[0]
}

func withVerhoeffShare(r *rand.Rand, d string) string {
	// Make a 12-digit distractor Verhoeff-valid with leading 2-9 at hardNegRate.
	if len(d) == 12 && r.Float64() < hardNegRate {
		return NewAadhaar(r)
	}
	return d
}

// rrn returns a UPI/IMPS retrieval reference number: YDDDHHSSSSSS.
func rrn(r *rand.Rand) string {
	y := baseTime.Year() % 10
	d := fmt.Sprintf("%d%03d%02d%06d", y, 1+r.Intn(365), r.Intn(24), r.Intn(1000000))
	return withVerhoeffShare(r, d)
}

// Fields is the catalogue. PII fields draw from the Person; distractors are
// independent values whose labels compete with PII labels for the same
// digit shapes.
var Fields = []*Field{
	{Kind: "aadhaar", Type: pii.Aadhaar, Weight: 10, Gen: func(r *rand.Rand, p *Person) string { return p.aadhaarFmt(r) },
		Labels: []string{"Aadhaar Number", "Aadhaar No", "Aadhaar No.", "Aadhaar", "Aadhar Number", "Aadhar No", "Aadhar", "Adhaar No", "Adhar Number",
			"UID", "UID No", "UIDAI No", "Aadhaar Card No", "Aadhaar Card Number", "Aadhar Card No.", "Aadhaar ID", "Aadhaar #", "e-Aadhaar No",
			"Aadhaar Num", "Aadhaar number (12 digit)", "Aadhaar No (as per card)", "Unique Identification Number", "AADHAAR", "Aadhaar Card",
			"Aadhar card no", "Customer Aadhaar", "Applicant Aadhaar No", "Aadhaar of applicant", "aadhaar_no", "aadhaarNumber", "aadhar_number", "uid_no", "AadhaarNo"}},
	{Kind: "aadhaar_masked", Type: "", Weight: 2, Gen: func(r *rand.Rand, p *Person) string { return p.maskedAadhaar(r) },
		Labels: []string{"Aadhaar Number", "Aadhaar No", "Aadhaar", "Aadhar No", "UID", "Masked Aadhaar", "Aadhaar (masked)", "aadhaar_no", "Aadhaar No (last 4)"}},
	{Kind: "vid", Type: pii.VID, Weight: 1, Gen: func(r *rand.Rand, p *Person) string { return groupDigits(p.VID, pick(r, []string{" ", "", " ", "-"}), 4) },
		Labels: []string{"VID", "Virtual ID", "Aadhaar VID", "VID No", "virtual_id", "Virtual Identification Number"}},
	{Kind: "pan", Type: pii.PAN, Weight: 6, Gen: func(r *rand.Rand, p *Person) string { return p.PAN },
		Labels: []string{"PAN", "PAN No", "PAN No.", "PAN Number", "PAN Card No", "PAN Card Number", "Permanent Account Number", "PAN Card", "pan_no", "panNumber", "PAN of applicant", "Income Tax PAN", "PAN (10 char)", "PAN ID"}},
	{Kind: "gstin", Type: pii.GSTIN, Weight: 3, Gen: func(r *rand.Rand, p *Person) string { return p.GSTIN },
		Labels: []string{"GSTIN", "GST No", "GST No.", "GST Number", "GSTIN No", "GST Registration No", "GST Identification Number", "GSTN", "gstin", "gst_no", "Supplier GSTIN", "Buyer GSTIN", "GSTIN/UIN", "GST Reg No"}},
	{Kind: "credit_card", Type: pii.CreditCard, Weight: 4, Gen: func(r *rand.Rand, p *Person) string { return p.cardFmt(r, NewCard(r)) },
		Labels: []string{"Card Number", "Card No", "Card No.", "Credit Card Number", "Credit Card No", "Debit Card Number", "Debit Card No", "Card", "card_number", "cardNo", "Visa Card No", "Mastercard No", "RuPay Card Number", "CC No", "Payment Card", "Card # "}},
	{Kind: "voter_id", Type: pii.VoterID, Weight: 2, Gen: func(r *rand.Rand, p *Person) string { return p.Voter },
		Labels: []string{"Voter ID", "Voter ID No", "Voter ID Number", "EPIC No", "EPIC Number", "Voter Card No", "Election Card No", "voter_id", "epicNo", "Voter Id Card Number"}},
	{Kind: "passport", Type: pii.Passport, Weight: 2, Gen: func(r *rand.Rand, p *Person) string { return p.Passport },
		Labels: []string{"Passport No", "Passport Number", "Passport No.", "Passport", "passport_no", "passportNumber", "Passport #", "PPT No"}},
	{Kind: "driving_licence", Type: pii.DrivingLicence, Weight: 2, Gen: func(r *rand.Rand, p *Person) string { return p.dlFmt(r) },
		Labels: []string{"Driving Licence No", "Driving License Number", "DL No", "DL Number", "DL No.", "Driving Licence", "dl_no", "licenseNumber", "Licence No"}},
	{Kind: "ifsc", Type: pii.IFSC, Weight: 4, Gen: func(r *rand.Rand, p *Person) string { return p.IFSC },
		Labels: []string{"IFSC", "IFSC Code", "IFS Code", "IFSC code", "ifsc_code", "ifsc", "Bank IFSC", "Branch IFSC", "IFSC:"}},
	{Kind: "mobile", Type: pii.Mobile, Weight: 8, Gen: func(r *rand.Rand, p *Person) string { return p.mobileFmt(r) },
		Labels: []string{"Mobile", "Mobile No", "Mobile No.", "Mobile Number", "Phone", "Phone No", "Phone Number", "Contact No", "Contact Number", "Contact", "Mob", "Mob No", "Cell", "WhatsApp No", "WhatsApp", "Registered Mobile", "RMN", "Tel", "mobile_no", "phoneNumber", "contact_number", "msisdn", "Alternate No", "Ph"}},
	{Kind: "email", Type: pii.Email, Weight: 5, Gen: func(r *rand.Rand, p *Person) string { return p.Email },
		Labels: []string{"Email", "Email ID", "E-mail", "Email Address", "Mail ID", "email", "email_id", "emailAddress", "E-Mail ID"}},
	{Kind: "upi_id", Type: pii.UPI, Weight: 3, Gen: func(r *rand.Rand, p *Person) string { return p.UPI },
		Labels: []string{"UPI ID", "UPI", "VPA", "UPI Handle", "upi_id", "vpa", "UPI Address", "Payee UPI ID", "Payer VPA"}},
	{Kind: "bank_account", Type: pii.BankAccount, Weight: 9, Gen: func(r *rand.Rand, p *Person) string { return p.Account },
		Labels: []string{"Account Number", "Account No", "Account No.", "A/c No", "A/C No.", "Acc No", "Acct No", "Bank Account Number", "Bank Account No", "Bank A/c", "Bank Acc", "Bank acc", "Savings Account No", "SB A/c No", "Current Account No",
			"Account", "Acc", "A/c", "account_no", "accountNumber", "acct_no", "bank_account", "Beneficiary Account No", "Beneficiary A/c", "Loan Account No", "Loan A/c No", "Salary Account", "Khata No", "Account Number (as per passbook)", "Acc. No", "Ac No"}},
	{Kind: "uan", Type: pii.UAN, Weight: 3, Gen: func(r *rand.Rand, p *Person) string { return p.UAN },
		Labels: []string{"UAN", "UAN No", "UAN Number", "Universal Account Number", "EPF UAN", "PF UAN", "uan_no", "uan", "EPF Account No", "PF Account Number", "PF No", "EPFO UAN"}},
	{Kind: "pincode", Type: pii.Pincode, Weight: 3, Gen: func(r *rand.Rand, p *Person) string { return p.Pin },
		Labels: []string{"PIN", "PIN Code", "Pincode", "Pin Code", "Postal Code", "ZIP", "Zip Code", "pincode", "pin_code", "postalCode", "PIN:"}},
	{Kind: "dob", Type: pii.DOB, Weight: 5, Gen: func(r *rand.Rand, p *Person) string { return p.dobFmt(r) },
		Labels: []string{"DOB", "Date of Birth", "D.O.B", "D.O.B.", "Birth Date", "dob", "date_of_birth", "dateOfBirth", "Born on", "DoB"}},
	{Kind: "name", Type: pii.Name, Weight: 9, Gen: func(r *rand.Rand, p *Person) string { return p.nameFmt(r) },
		Labels: []string{"Name", "Full Name", "Customer Name", "Applicant Name", "Account Holder Name", "Holder Name", "Cardholder Name", "Beneficiary Name", "Nominee Name", "Employee Name", "Name of Applicant", "Name as per Aadhaar", "Name as per PAN", "name", "full_name", "customerName", "applicant_name", "Contact Person", "Proprietor Name", "Student Name", "Patient Name"}},
	{Kind: "father_name", Type: pii.Name, Weight: 2, Gen: func(r *rand.Rand, p *Person) string { return p.Father },
		Labels: []string{"Father's Name", "Father Name", "Fathers Name", "Father/Husband Name", "Guardian Name", "father_name", "fatherName", "S/o", "D/o", "W/o"}},
	{Kind: "address", Type: pii.Address, Weight: 6, Gen: func(r *rand.Rand, p *Person) string { return p.addressFmt(r) },
		Labels: []string{"Address", "Residential Address", "Permanent Address", "Correspondence Address", "Communication Address", "Present Address", "Current Address", "Billing Address", "Shipping Address", "Delivery Address", "Registered Address", "address", "residential_address", "permanentAddress", "Addr", "Add.", "Residing at", "Address for communication", "Office Address", "Home Address"}},

	// ----- non-PII distractors ------------------------------------------------
	{Kind: "rrn", Type: "", Weight: 8, Gen: func(r *rand.Rand, p *Person) string { return rrn(r) },
		Labels: []string{"RRN", "UPI Ref No", "UPI Ref", "UPI Reference No", "UPI Txn ID", "IMPS Ref No", "IMPS Reference", "IMPS Ref", "Ref No", "Reference No", "Reference Number", "Txn Ref", "Transaction Reference", "Retrieval Reference Number", "RRN No", "rrn", "upi_ref_no", "Bank Ref No", "Bank Reference No", "UTR", "UTR No", "NEFT Ref", "Txn Ref No"}},
	{Kind: "txn_id", Type: "", Weight: 7, Gen: func(r *rand.Rand, p *Person) string {
		switch r.Intn(5) {
		case 0:
			return "T" + digits(r, 16) // PhonePe style
		case 1:
			return pick(r, []string{"TXN", "PAY", "TR", "PI"}) + digits(r, 10+r.Intn(4))
		case 2:
			return withVerhoeffShare(r, digits(r, 12))
		case 3:
			return digits(r, 10+r.Intn(5))
		default:
			return strings.ToUpper(letters(r, 2)) + digits(r, 14)
		}
	}, Labels: []string{"Transaction ID", "Txn ID", "Txn No", "Transaction No", "Transaction Number", "TXN ID", "Payment ID", "Payment Ref", "Payment Reference", "Trans ID", "txn_id", "transactionId", "paymentId", "Txn#", "Transaction Ref No", "Pay ID", "Refund ID", "Settlement ID", "Auth Code", "ARN"}},
	{Kind: "order_id", Type: "", Weight: 6, Gen: func(r *rand.Rand, p *Person) string {
		switch r.Intn(5) {
		case 0:
			return "OD" + digits(r, 18)
		case 1:
			return fmt.Sprintf("%03d-%07d-%07d", 100+r.Intn(900), r.Intn(10000000), r.Intn(10000000))
		case 2:
			return withVerhoeffShare(r, digits(r, 12))
		case 3:
			return "ORD" + digits(r, 8+r.Intn(4))
		default:
			return digits(r, 8+r.Intn(6))
		}
	}, Labels: []string{"Order ID", "Order No", "Order Number", "Order #", "Order", "order_id", "orderId", "Ord No", "PO Number", "PO No", "Purchase Order No", "Sales Order No", "Suborder ID", "Cart ID", "Order Ref"}},
	{Kind: "invoice", Type: "", Weight: 4, Gen: func(r *rand.Rand, p *Person) string {
		switch r.Intn(4) {
		case 0:
			return fmt.Sprintf("INV-%d-%06d", 2020+r.Intn(7), r.Intn(1000000))
		case 1:
			return fmt.Sprintf("INV/%02d-%02d/%05d", 20+r.Intn(7), 21+r.Intn(7), r.Intn(100000))
		case 2:
			return withVerhoeffShare(r, digits(r, 12))
		default:
			return digits(r, 6+r.Intn(5))
		}
	}, Labels: []string{"Invoice No", "Invoice Number", "Invoice #", "Invoice ID", "Inv No", "Bill No", "Bill Number", "Receipt No", "Receipt Number", "Challan No", "Voucher No", "invoice_no", "invoiceNumber", "Tax Invoice No", "Bill ID", "E-way Bill No"}},
	{Kind: "tracking", Type: "", Weight: 4, Gen: func(r *rand.Rand, p *Person) string {
		switch r.Intn(3) {
		case 0:
			return withVerhoeffShare(r, digits(r, 12))
		case 1:
			return digits(r, 10+r.Intn(5))
		default:
			return strings.ToUpper(letters(r, 2)) + digits(r, 9) + "IN"
		}
	}, Labels: []string{"AWB", "AWB No", "AWB Number", "Tracking ID", "Tracking No", "Tracking Number", "Consignment No", "Consignment Number", "Shipment ID", "Docket No", "LR No", "Waybill No", "awb_no", "trackingId", "Courier Ref"}},
	{Kind: "customer_id", Type: "", Weight: 6, Gen: func(r *rand.Rand, p *Person) string {
		switch r.Intn(5) {
		case 0:
			return withVerhoeffShare(r, digits(r, 12))
		case 1:
			return digits(r, 8+r.Intn(4))
		case 2:
			return "CUST" + digits(r, 7)
		case 3:
			return "EMP" + digits(r, 5)
		default:
			return digits(r, 11) // SBI CIF-like
		}
	}, Labels: []string{"Customer ID", "Cust ID", "Customer No", "Customer Number", "Customer Code", "CIF", "CIF No", "CIF Number", "CRN", "Client ID", "Member ID", "Membership No", "Employee ID", "Emp ID", "Emp Code", "Employee Code", "User ID", "Login ID", "Subscriber ID", "Agent Code", "Merchant ID", "customer_id", "customerId", "cif_no", "emp_id", "userId", "Account ID", "Profile ID", "Application No", "Application Number", "Registration No", "Reg No", "Roll No", "Ack No", "Acknowledgement No"}},
	{Kind: "policy", Type: "", Weight: 3, Gen: func(r *rand.Rand, p *Person) string {
		switch r.Intn(3) {
		case 0:
			return withVerhoeffShare(r, digits(r, 12))
		case 1:
			return digits(r, 9)
		default:
			return fmt.Sprintf("%d/%d/%d", 10+r.Intn(90), r.Intn(1000000), 2015+r.Intn(12))
		}
	}, Labels: []string{"Policy No", "Policy Number", "Policy #", "Policy", "Proposal No", "Claim No", "Claim Number", "Claim ID", "Folio No", "Folio Number", "policy_no", "policyNumber", "Certificate No", "LIC Policy No", "Insurance Policy No"}},
	{Kind: "ticket", Type: "", Weight: 5, Gen: func(r *rand.Rand, p *Person) string {
		switch r.Intn(5) {
		case 0:
			return "INC" + digits(r, 7)
		case 1:
			return "SR" + digits(r, 8+r.Intn(3))
		case 2:
			return withVerhoeffShare(r, digits(r, 12))
		case 3:
			return digits(r, 7+r.Intn(4))
		default:
			return fmt.Sprintf("%08x-%04x", r.Uint32(), r.Intn(65536))
		}
	}, Labels: []string{"Ticket No", "Ticket Number", "Ticket ID", "Ticket #", "Case ID", "Case No", "Incident No", "Incident ID", "SR No", "SR Number", "Service Request No", "Request ID", "Req ID", "Complaint No", "Complaint Number", "Grievance ID", "ticket_id", "requestId", "caseId", "Job ID", "Batch ID", "Session ID", "Trace ID", "Correlation ID", "Message ID", "Record ID", "Seq No", "Sl No", "Sr No"}},
	{Kind: "pran", Type: "", Weight: 2, Gen: func(r *rand.Rand, p *Person) string { return "11" + digits(r, 10) },
		Labels: []string{"PRAN", "PRAN No", "PRAN Number", "NPS PRAN", "pran_no", "NPS Account No"}},
	{Kind: "ppo", Type: "", Weight: 2, Gen: func(r *rand.Rand, p *Person) string { return withVerhoeffShare(r, digits(r, 12)) },
		Labels: []string{"PPO No", "PPO Number", "PPO", "Pension Payment Order No", "ppo_no", "Pension No"}},
	{Kind: "apaar", Type: "", Weight: 1, Gen: func(r *rand.Rand, p *Person) string { return withVerhoeffShare(r, digits(r, 12)) },
		Labels: []string{"APAAR ID", "APAAR", "ABC ID", "apaar_id", "APAAR Number"}},
	{Kind: "consumer_no", Type: "", Weight: 4, Gen: func(r *rand.Rand, p *Person) string {
		switch r.Intn(3) {
		case 0:
			return withVerhoeffShare(r, digits(r, 12))
		case 1:
			return digits(r, 10+r.Intn(3))
		default:
			return digits(r, 9)
		}
	}, Labels: []string{"Consumer No", "Consumer Number", "Consumer ID", "CA Number", "CA No", "K Number", "K No", "Service No", "Connection No", "Meter No", "LPG ID", "LPG Consumer No", "BP Number", "RR Number", "IVRS No", "consumer_no", "Subscriber No", "Electricity Account No", "Gas Connection No"}},
	{Kind: "device_id", Type: "", Weight: 3, Gen: func(r *rand.Rand, p *Person) string {
		switch r.Intn(4) {
		case 0:
			return NewIMEI(r)
		case 1:
			return strings.ToUpper(letters(r, 3)) + digits(r, 9)
		case 2:
			return withVerhoeffShare(r, digits(r, 12))
		default:
			return fmt.Sprintf("%02X:%02X:%02X:%02X:%02X:%02X", r.Intn(256), r.Intn(256), r.Intn(256), r.Intn(256), r.Intn(256), r.Intn(256))
		}
	}, Labels: []string{"IMEI", "IMEI No", "IMEI Number", "Device ID", "Serial No", "Serial Number", "SN", "MAC", "MAC Address", "SIM No", "ICCID", "device_id", "imei", "serialNo", "Asset ID", "Asset Tag", "Model No"}},
	{Kind: "pnr", Type: "", Weight: 2, Gen: func(r *rand.Rand, p *Person) string {
		if r.Intn(2) == 0 {
			return digits(r, 10)
		}
		return strings.ToUpper(letters(r, 6))
	}, Labels: []string{"PNR", "PNR No", "PNR Number", "Booking ID", "Booking Ref", "Booking Reference", "Reservation No", "Confirmation No", "pnr", "bookingId"}},
	{Kind: "timestamp", Type: "", Weight: 4, Gen: func(r *rand.Rand, p *Person) string {
		t := baseTime.Add(-time.Duration(r.Intn(100000)) * time.Minute)
		switch r.Intn(4) {
		case 0:
			return fmt.Sprint(t.UnixMilli())
		case 1:
			return fmt.Sprint(t.Unix())
		case 2:
			return t.Format("2006-01-02T15:04:05Z")
		default:
			return t.Format("02/01/2006 15:04")
		}
	}, Labels: []string{"Timestamp", "TS", "Epoch", "Created At", "Updated At", "Event Time", "Logged At", "created_at", "updatedAt", "ts", "Time", "Date", "Generated At", "Expires At", "Valid Till"}},
	{Kind: "amount", Type: "", Weight: 6, Gen: func(r *rand.Rand, p *Person) string {
		v := r.Intn(5000000)
		switch r.Intn(5) {
		case 0:
			return "Rs. " + indianGrouping(v) + ".00"
		case 1:
			return "INR " + indianGrouping(v)
		case 2:
			return "₹" + indianGrouping(v)
		case 3:
			return fmt.Sprintf("%d.%02d", v, r.Intn(100))
		default:
			return fmt.Sprint(v)
		}
	}, Labels: []string{"Amount", "Amt", "Total", "Total Amount", "Balance", "Available Balance", "Net Amount", "Premium", "EMI", "Salary", "Price", "Fee", "Charges", "amount", "totalAmount", "balance", "Outstanding", "Paid", "Due Amount", "MRP", "Invoice Amount"}},
	{Kind: "cheque", Type: "", Weight: 2, Gen: func(r *rand.Rand, p *Person) string {
		if r.Intn(3) == 0 {
			return digits(r, 9) // MICR
		}
		return digits(r, 6)
	}, Labels: []string{"Cheque No", "Cheque Number", "Chq No", "Cheque", "DD No", "MICR", "MICR Code", "cheque_no", "Check No"}},
	{Kind: "vehicle", Type: "", Weight: 2, Gen: func(r *rand.Rand, p *Person) string {
		rto := rtoCodes[p.State]
		if rto == "" {
			rto = "KA"
		}
		return fmt.Sprintf("%s%02d%s%04d", rto, 1+r.Intn(60), letters(r, 2), r.Intn(10000))
	}, Labels: []string{"Vehicle No", "Vehicle Number", "Registration No", "Regn No", "RC No", "vehicle_no", "Vehicle Reg No", "FASTag Vehicle No"}},
	{Kind: "aadhaar_eid", Type: "", Weight: 1, Gen: func(r *rand.Rand, p *Person) string { return fmt.Sprintf("%04d/%05d/%05d", r.Intn(10000), r.Intn(100000), r.Intn(100000)) },
		Labels: []string{"Enrolment No", "Enrolment ID", "EID", "Aadhaar Enrolment No", "Enrollment Number", "enrolment_id"}},
	{Kind: "generic_id", Type: "", Weight: 5, Gen: func(r *rand.Rand, p *Person) string {
		switch r.Intn(4) {
		case 0:
			return withVerhoeffShare(r, digits(r, 12))
		case 1:
			return digits(r, 6+r.Intn(10))
		case 2:
			return fmt.Sprintf("%08x", r.Uint32())
		default:
			return fmt.Sprint(r.Intn(100000))
		}
	}, Labels: []string{"ID", "Id", "id", "Number", "No", "No.", "Ref", "Code", "Key", "Value", "Identifier", "Record", "Field", "Entry", "Item", "Num"}},
	{Kind: "user_id", Type: "", Weight: 4, Gen: func(r *rand.Rand, p *Person) string {
		switch r.Intn(5) {
		case 0:
			return withVerhoeffShare(r, digits(r, 12))
		case 1:
			return strings.ToLower(p.First) + "." + strings.ToLower(p.Last)
		case 2:
			return fmt.Sprintf("u%d", r.Intn(1000000))
		default:
			return digits(r, 4+r.Intn(6))
		}
	}, Labels: []string{"uid", "UID", "user id", "userId", "user_id", "User ID", "login id", "username", "user", "principal", "actor", "subject id"}},
	{Kind: "count", Type: "", Weight: 3, Gen: func(r *rand.Rand, p *Person) string { return fmt.Sprint(r.Intn(1000000)) },
		Labels: []string{"Count", "Total Records", "Records", "Rows", "Qty", "Quantity", "Processed", "Failed", "Pending", "Units", "Pages", "Items"}},
	{Kind: "otp", Type: "", Weight: 3, Gen: func(r *rand.Rand, p *Person) string { return digits(r, 4+2*r.Intn(2)) },
		Labels: []string{"OTP", "One Time Password", "Verification Code", "Code", "PIN", "MPIN", "Passcode", "Security Code", "otp"}},
	{Kind: "percent", Type: "", Weight: 2, Gen: func(r *rand.Rand, p *Person) string { return fmt.Sprintf("%d.%d%%", r.Intn(100), r.Intn(10)) },
		Labels: []string{"Interest Rate", "Rate", "ROI", "Discount", "GST Rate", "Tax Rate", "Growth", "Share"}},
	{Kind: "dummy", Type: "", Weight: 2, Gen: func(r *rand.Rand, p *Person) string {
		// test fixtures: repeated / sequential digits with a valid check digit
		var base string
		switch r.Intn(4) {
		case 0:
			base = strings.Repeat(string(byte('2'+r.Intn(8))), 11)
		case 1:
			base = "23456789012"
		case 2:
			base = strings.Repeat("1234", 2) + "123"
			base = string(byte('2'+r.Intn(8))) + base[1:]
		default:
			base = "98765432109"
		}
		d := base + string(checksum.VerhoeffCheckDigit(base))
		if r.Intn(2) == 0 {
			return groupDigits(d, " ", 4)
		}
		return d
	}, Labels: []string{"Aadhaar", "Aadhaar No", "aadhaar_no", "test_aadhaar", "sample aadhaar", "AADHAAR_TEST", "dummy aadhaar", "aadhaar (sample)", "Aadhaar Number"}},
}

// FieldByKind indexes Fields.
var FieldByKind = func() map[string]*Field {
	m := map[string]*Field{}
	for _, f := range Fields {
		m[f.Kind] = f
	}
	return m
}()

// indianGrouping formats 1234567 as 12,34,567.
func indianGrouping(v int) string {
	s := fmt.Sprint(v)
	if len(s) <= 3 {
		return s
	}
	last3 := s[len(s)-3:]
	rest := s[:len(s)-3]
	var parts []string
	for len(rest) > 2 {
		parts = append([]string{rest[len(rest)-2:]}, parts...)
		rest = rest[:len(rest)-2]
	}
	if rest != "" {
		parts = append([]string{rest}, parts...)
	}
	return strings.Join(parts, ",") + "," + last3
}

// IsPIIKind reports whether a field kind is PII.
func IsPIIKind(kind string) bool {
	f := FieldByKind[kind]
	return f != nil && f.Type != ""
}

// labelKindOf returns the labels.Kind that names a field kind (for
// consistency checks between the generator and the catalogue).
func labelKindOf(kind string) labels.Kind {
	switch kind {
	case "father_name":
		return labels.KName
	case "rrn", "txn_id":
		return labels.KTxn
	case "dummy":
		return labels.KAadhaar
	case "count", "otp", "percent":
		return labels.KGeneric
	}
	return labels.Kind(kind)
}
