// Package synth generates synthetic Indian-context documents with labelled
// PII spans and labelled look-alike distractors, in the data shapes the
// product scans (prose, forms, CSV, JSON, logs, code, OCR output). It is the
// only training/evaluation data available for this POC, so it is built to be
// adversarial: it deliberately places competing identifiers next to PII and
// makes a share of non-PII 12-digit numbers pass the Verhoeff check.
package synth

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/techiek77-ui/poc1/pkg/pii/checksum"
	"github.com/techiek77-ui/poc1/pkg/pii/gazetteer"
)

// Person is a coherent synthetic identity used across one document.
type Person struct {
	First, Last, Full, Father string
	Gender                    string
	DOB                       time.Time
	Mobile                    string // 10 digits
	Email, UPI                string
	Aadhaar, VID              string // digits only
	PAN, Voter, Passport, DL  string
	BankName, BankCode        string
	Account, IFSC, UAN        string
	House, Street, Locality   string
	City, State, Pin          string
	Company, GSTIN            string
	CustomerID                string
}

type bank struct {
	name, code string
	acctLen    int
}

var banks = []bank{
	{"State Bank of India", "SBIN", 11}, {"HDFC Bank", "HDFC", 14}, {"ICICI Bank", "ICIC", 12}, {"Axis Bank", "UTIB", 15},
	{"Punjab National Bank", "PUNB", 16}, {"Kotak Mahindra Bank", "KKBK", 10}, {"Bank of Baroda", "BARB", 14},
	{"Canara Bank", "CNRB", 13}, {"Union Bank of India", "UBIN", 15}, {"IDBI Bank", "IBKL", 16}, {"Yes Bank", "YESB", 15},
	{"IndusInd Bank", "INDB", 12}, {"Federal Bank", "FDRL", 14}, {"Indian Bank", "IDIB", 9}, {"Bank of India", "BKID", 15},
	{"Central Bank of India", "CBIN", 10}, {"Indian Overseas Bank", "IOBA", 15}, {"UCO Bank", "UCBA", 14},
	{"Bank of Maharashtra", "MAHB", 11}, {"AU Small Finance Bank", "AUBL", 16}, {"Paytm Payments Bank", "PYTM", 12},
	{"Airtel Payments Bank", "AIRP", 12}, {"Karnataka Bank", "KARB", 16}, {"South Indian Bank", "SIBL", 16},
	{"RBL Bank", "RATN", 12}, {"IDFC First Bank", "IDFB", 11}, {"Bandhan Bank", "BDBL", 15}, {"DBS Bank", "DBSS", 12},
}

var companies = []string{
	"Infotech Solutions Pvt Ltd", "Shree Ganesh Traders", "Apex Logistics", "Sunrise Textiles", "Mahalaxmi Enterprises",
	"Nova Pharma Ltd", "Bharat Steel Works", "Greenleaf Organics", "Omkar Constructions", "Delta Motors", "Krishna Agro",
	"Vardhman Exports", "BlueSky Travels", "Prime Finserv", "Tech Mahadev Systems", "Saraswati Printers", "Ganga Foods",
	"Royal Jewellers", "UrbanNest Realty", "Pioneer Chemicals", "Zenith Electronics", "Metro Cabs", "Lotus Hospitals",
}

var emailDomains = []string{"gmail.com", "gmail.com", "gmail.com", "yahoo.co.in", "yahoo.com", "outlook.com", "hotmail.com", "rediffmail.com", "icloud.com", "protonmail.com"}
var upiHandles = []string{"okaxis", "oksbi", "okhdfcbank", "okicici", "ybl", "ibl", "axl", "paytm", "upi", "apl", "fbl", "idfcbank", "kotak", "airtel", "jio"}

var stateCodes = map[string]string{
	"Andhra Pradesh": "37", "Arunachal Pradesh": "12", "Assam": "18", "Bihar": "10", "Chhattisgarh": "22", "Goa": "30",
	"Gujarat": "24", "Haryana": "06", "Himachal Pradesh": "02", "Jharkhand": "20", "Karnataka": "29", "Kerala": "32",
	"Madhya Pradesh": "23", "Maharashtra": "27", "Manipur": "14", "Meghalaya": "17", "Mizoram": "15", "Nagaland": "13",
	"Odisha": "21", "Punjab": "03", "Rajasthan": "08", "Sikkim": "11", "Tamil Nadu": "33", "Telangana": "36", "Tripura": "16",
	"Uttar Pradesh": "09", "Uttarakhand": "05", "West Bengal": "19", "Delhi": "07", "Jammu and Kashmir": "01", "Ladakh": "38",
	"Puducherry": "34", "Chandigarh": "04",
}

var rtoCodes = map[string]string{
	"Maharashtra": "MH", "Karnataka": "KA", "Delhi": "DL", "Tamil Nadu": "TN", "Telangana": "TS", "Gujarat": "GJ",
	"Uttar Pradesh": "UP", "West Bengal": "WB", "Rajasthan": "RJ", "Kerala": "KL", "Punjab": "PB", "Haryana": "HR",
	"Madhya Pradesh": "MP", "Bihar": "BR", "Andhra Pradesh": "AP", "Odisha": "OD", "Assam": "AS", "Jharkhand": "JH",
	"Chhattisgarh": "CG", "Uttarakhand": "UK", "Goa": "GA", "Himachal Pradesh": "HP", "Chandigarh": "CH", "Puducherry": "PY",
}

func pick[T any](r *rand.Rand, xs []T) T { return xs[r.Intn(len(xs))] }

func digits(r *rand.Rand, n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte('0' + r.Intn(10))
	}
	return string(b)
}

func letters(r *rand.Rand, n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte('A' + r.Intn(26))
	}
	return string(b)
}

// NewAadhaar returns a Verhoeff-valid 12-digit number starting with 2-9.
func NewAadhaar(r *rand.Rand) string {
	p := string(byte('2'+r.Intn(8))) + digits(r, 10)
	return p + string(checksum.VerhoeffCheckDigit(p))
}

// NewVID returns a Verhoeff-valid 16-digit virtual ID.
func NewVID(r *rand.Rand) string {
	p := string(byte('2'+r.Intn(8))) + digits(r, 14)
	return p + string(checksum.VerhoeffCheckDigit(p))
}

// NewPAN returns a structurally valid PAN. typ is the 4th character.
func NewPAN(r *rand.Rand, typ byte, surname string) string {
	s := byte('A' + r.Intn(26))
	if surname != "" {
		s = strings.ToUpper(surname)[0]
	}
	return letters(r, 3) + string(typ) + string(s) + digits(r, 4) + letters(r, 1)
}

// NewGSTIN returns a GSTIN with a valid check character.
func NewGSTIN(r *rand.Rand, state string) string {
	sc, ok := stateCodes[state]
	if !ok {
		sc = fmt.Sprintf("%02d", 1+r.Intn(36))
	}
	typ := pick(r, []byte{'C', 'C', 'F', 'P', 'H', 'A', 'T'})
	p := sc + NewPAN(r, typ, "") + string(pick(r, []byte("123456789"))) + "Z"
	return p + string(checksum.GSTINCheckChar(p))
}

// NewCard returns a Luhn-valid card number with a real network prefix.
func NewCard(r *rand.Rand) string {
	var p string
	switch r.Intn(10) {
	case 0, 1, 2, 3:
		p = "4" + digits(r, 14)
	case 4, 5:
		p = string(byte('5'+0)) + string(byte('1'+r.Intn(5))) + digits(r, 13)
	case 6:
		p = fmt.Sprintf("%d", 2221+r.Intn(500)) + digits(r, 11)
	case 7:
		p = pick(r, []string{"34", "37"}) + digits(r, 12) // Amex 15
	default:
		p = pick(r, []string{"60", "65", "81", "82", "508"})
		p += digits(r, 15-len(p))
	}
	return p + string(checksum.LuhnCheckDigit(p))
}

// NewIMEI returns a Luhn-valid 15-digit IMEI.
func NewIMEI(r *rand.Rand) string {
	p := pick(r, []string{"35", "86", "01", "49"}) + digits(r, 12)
	return p + string(checksum.LuhnCheckDigit(p))
}

// NewPerson builds a coherent identity.
func NewPerson(r *rand.Rand) *Person {
	p := &Person{}
	p.First = pick(r, gazetteer.FirstNames)
	p.Last = pick(r, gazetteer.Surnames)
	p.Full = p.First + " " + p.Last
	p.Father = pick(r, gazetteer.FirstNames) + " " + p.Last
	p.Gender = pick(r, []string{"Male", "Female"})
	p.DOB = time.Date(1955+r.Intn(50), time.Month(1+r.Intn(12)), 1+r.Intn(28), 0, 0, 0, 0, time.UTC)
	p.Mobile = string(byte('6'+r.Intn(4))) + digits(r, 9)
	sep := pick(r, []string{".", "", "_", ""})
	p.Email = strings.ToLower(p.First) + sep + strings.ToLower(p.Last) + pick(r, []string{"", "", fmt.Sprint(r.Intn(99)), fmt.Sprint(1970 + r.Intn(40))}) + "@" + pick(r, emailDomains)
	if r.Intn(3) == 0 {
		p.UPI = p.Mobile + "@" + pick(r, upiHandles)
	} else {
		p.UPI = strings.ToLower(p.First) + strings.ToLower(p.Last)[:1+r.Intn(len(p.Last)-1)] + "@" + pick(r, upiHandles)
	}
	p.Aadhaar = NewAadhaar(r)
	p.VID = NewVID(r)
	p.PAN = NewPAN(r, 'P', p.Last)
	p.Voter = letters(r, 3) + digits(r, 7)
	p.Passport = string(pick(r, []byte("ABCDEFGHJKLMNPRSTUVWY"))) + string(byte('1'+r.Intn(9))) + digits(r, 6)
	p.State = pick(r, gazetteer.States)
	rto := rtoCodes[p.State]
	if rto == "" {
		rto = "MH"
	}
	p.DL = fmt.Sprintf("%s%02d%d%07d", rto, 1+r.Intn(50), 1995+r.Intn(28), r.Intn(10000000))
	b := pick(r, banks)
	p.BankName, p.BankCode = b.name, b.code
	p.Account = newAccount(r, b.acctLen)
	p.IFSC = b.code + "0" + digits(r, 6)
	if r.Intn(5) == 0 {
		p.IFSC = b.code + "0" + letters(r, 2) + digits(r, 4)
	}
	p.UAN = "10" + digits(r, 10)
	p.House = pick(r, []string{
		fmt.Sprintf("%d", 1+r.Intn(400)), fmt.Sprintf("No. %d", 1+r.Intn(400)), fmt.Sprintf("#%d", 1+r.Intn(400)),
		fmt.Sprintf("Flat %d", 101+r.Intn(1500)), fmt.Sprintf("Flat No. %d", 101+r.Intn(1500)), fmt.Sprintf("%d/%d", 1+r.Intn(99), 1+r.Intn(30)),
		fmt.Sprintf("Plot %d", 1+r.Intn(300)), fmt.Sprintf("H.No. %d-%d-%d", 1+r.Intn(20), 1+r.Intn(99), 1+r.Intn(999)),
		fmt.Sprintf("%c-%d", 'A'+r.Intn(8), 1+r.Intn(300)), fmt.Sprintf("Door No %d", 1+r.Intn(200)),
	})
	p.Street = pick(r, []string{
		pick(r, []string{"MG", "Station", "Nehru", "Gandhi", "Link", "Ring", "Temple", "Market", "Church", "Hill", "Lake", "Park", "College", "Hospital", "Mill", "Canal", "Palace", "Fort"}) + " " + pick(r, []string{"Road", "Road", "Street", "Lane", "Marg"}),
		fmt.Sprintf("%d%s Cross", 1+r.Intn(20), pick(r, []string{"st", "nd", "rd", "th"})),
		fmt.Sprintf("%d%s Main", 1+r.Intn(20), pick(r, []string{"st", "nd", "rd", "th"})),
		pick(r, gazetteer.FirstNames) + " " + pick(r, gazetteer.BuildingWords),
		pick(r, []string{"Sector", "Phase", "Block"}) + " " + fmt.Sprint(1+r.Intn(60)),
		// streets named after people: "Mohammed Ali Road", "Lal Bahadur Shastri Marg"
		pick(r, gazetteer.FirstNames) + " " + pick(r, gazetteer.Surnames) + " " + pick(r, []string{"Road", "Marg", "Street", "Nagar", "Lane", "Chowk"}),
	})
	p.Locality = pick(r, gazetteer.Localities)
	p.City = pick(r, gazetteer.Cities)
	p.Pin = string(byte('1'+r.Intn(8))) + digits(r, 5)
	p.Company = pick(r, companies)
	p.GSTIN = NewGSTIN(r, p.State)
	p.CustomerID = digits(r, 8+r.Intn(3))
	return p
}

// newAccount makes a bank account number of the given length. Accounts
// often carry leading zeros; a share of 12-digit ones are made to pass the
// Verhoeff check with a leading 2-9 so that they are indistinguishable from
// Aadhaar numbers by shape alone (the hard negative).
func newAccount(r *rand.Rand, n int) string {
	a := digits(r, n)
	if r.Intn(3) == 0 {
		a = strings.Repeat("0", 1+r.Intn(3)) + a[1+r.Intn(3)-0:]
		for len(a) < n {
			a += digits(r, 1)
		}
		a = a[:n]
	}
	if n == 12 && r.Float64() < hardNegRate {
		return NewAadhaar(r)
	}
	return a
}

// hardNegRate is the share of 12-digit non-Aadhaar values that are forced to
// be Verhoeff-valid with a leading 2-9. Random 12-digit strings pass at ~10%
// (and ~8% after the first-digit rule); the higher rate here makes the
// evaluation harder, not easier.
var hardNegRate = 0.35

// SetHardNegativeRate overrides hardNegRate (0..1).
func SetHardNegativeRate(f float64) { hardNegRate = f }

func (p *Person) dobFmt(r *rand.Rand) string {
	switch r.Intn(6) {
	case 0:
		return p.DOB.Format("02/01/2006")
	case 1:
		return p.DOB.Format("02-01-2006")
	case 2:
		return p.DOB.Format("2006-01-02")
	case 3:
		return p.DOB.Format("2 Jan 2006")
	case 4:
		return p.DOB.Format("02.01.2006")
	default:
		return p.DOB.Format("02 January 2006")
	}
}

// groupDigits formats a digit string in groups of 4 with sep.
func groupDigits(d string, sep string, group int) string {
	var parts []string
	for i := 0; i < len(d); i += group {
		e := min(len(d), i+group)
		parts = append(parts, d[i:e])
	}
	return strings.Join(parts, sep)
}

func (p *Person) aadhaarFmt(r *rand.Rand) string {
	switch r.Intn(9) {
	case 0, 1, 2, 3:
		return p.Aadhaar
	case 4, 5, 6, 7:
		return groupDigits(p.Aadhaar, " ", 4)
	default:
		return groupDigits(p.Aadhaar, "-", 4)
	}
}

// maskedAadhaar is what documents print after masking; it is not PII.
func (p *Person) maskedAadhaar(r *rand.Rand) string {
	return pick(r, []string{"XXXX XXXX ", "XXXX-XXXX-", "********", "xxxx xxxx "}) + p.Aadhaar[8:]
}

func (p *Person) mobileFmt(r *rand.Rand) string {
	switch r.Intn(10) {
	case 0, 1, 2, 3:
		return p.Mobile
	case 4, 5:
		return "+91 " + p.Mobile[:5] + " " + p.Mobile[5:]
	case 6:
		return "+91-" + p.Mobile
	case 7:
		return "+91" + p.Mobile
	case 8:
		return "91" + p.Mobile // 12 contiguous digits: collides with the Aadhaar shape
	default:
		return "0" + p.Mobile
	}
}

func (p *Person) cardFmt(r *rand.Rand, card string) string {
	switch r.Intn(5) {
	case 0, 1:
		return card
	case 2, 3:
		if len(card) == 15 {
			return card[:4] + " " + card[4:10] + " " + card[10:]
		}
		return groupDigits(card, " ", 4)
	default:
		return groupDigits(card, "-", 4)
	}
}

func (p *Person) nameFmt(r *rand.Rand) string {
	switch r.Intn(8) {
	case 0, 1, 2:
		return p.Full
	case 3:
		return strings.ToUpper(p.Full)
	case 4:
		return p.First[:1] + ". " + p.Last
	case 5:
		return p.Last + " " + p.First
	case 6:
		return p.First + " " + pick(r, gazetteer.FirstNames) + " " + p.Last
	default:
		return p.First
	}
}

func (p *Person) addressFmt(r *rand.Rand) string {
	switch r.Intn(6) {
	case 0:
		return fmt.Sprintf("%s, %s, %s, %s - %s", p.House, p.Street, p.Locality, p.City, p.Pin)
	case 1:
		return fmt.Sprintf("%s, %s, %s, %s, %s %s", p.House, p.Street, p.Locality, p.City, p.State, p.Pin)
	case 2:
		return fmt.Sprintf("%s %s, %s, %s %s", p.House, p.Street, p.City, p.State, p.Pin)
	case 3:
		return fmt.Sprintf("%s, %s, Near %s, %s, %s, PIN %s", p.House, p.Street, pick(r, []string{"Bus Stand", "Railway Station", "Shiv Temple", "City Mall", "Govt School", "Post Office", "Metro Station", "Bank of Baroda"}), p.Locality, p.City, p.Pin)
	case 4:
		return fmt.Sprintf("%s, %s, %s, Dist %s, %s-%s", p.House, p.Street, p.Locality, p.City, p.State, p.Pin)
	default:
		return fmt.Sprintf("%s %s %s %s %s", p.House, p.Street, p.Locality, p.City, p.Pin)
	}
}

func (p *Person) dlFmt(r *rand.Rand) string {
	d := p.DL
	switch r.Intn(4) {
	case 0:
		return d
	case 1:
		return d[:4] + " " + d[4:]
	case 2:
		return d[:2] + "-" + d[2:4] + "-" + d[4:8] + "-" + d[8:]
	default:
		return d[:4] + " " + d[4:8] + " " + d[8:]
	}
}
