// Package gazetteer holds word lists used both to detect names/addresses in
// the baseline (pattern + dictionary, the "other way" when no NER model is
// available) and to generate realistic synthetic Indian names and addresses.
package gazetteer

import "strings"

var FirstNames = []string{
	"Aarav", "Aditya", "Ajay", "Akash", "Akhil", "Amit", "Anand", "Anil", "Anjali", "Ankit", "Ankita", "Anuj",
	"Anusha", "Arjun", "Arun", "Ashok", "Ashish", "Ayesha", "Bhavna", "Bharat", "Chetan", "Deepa", "Deepak",
	"Dinesh", "Divya", "Farhan", "Fatima", "Gaurav", "Geeta", "Girish", "Gopal", "Harish", "Hema", "Imran",
	"Irfan", "Jaya", "Jyoti", "Kavita", "Karan", "Karthik", "Kiran", "Kishore", "Komal", "Krishna", "Kunal",
	"Lakshmi", "Lalit", "Madhu", "Mahesh", "Manish", "Manoj", "Meena", "Meera", "Mohan", "Mohammed", "Mohit",
	"Mukesh", "Naveen", "Neha", "Nikhil", "Nisha", "Nitin", "Pallavi", "Pankaj", "Pooja", "Prakash", "Pradeep",
	"Pranav", "Prasad", "Praveen", "Priya", "Priyanka", "Rahul", "Raj", "Rajesh", "Rakesh", "Ramesh", "Rani",
	"Ravi", "Rekha", "Ritu", "Rohit", "Rohan", "Sachin", "Sagar", "Sameer", "Sandeep", "Sanjay", "Santosh",
	"Sarita", "Saurabh", "Seema", "Shalini", "Shankar", "Sharad", "Shilpa", "Shreya", "Shruti", "Shyam",
	"Siddharth", "Smita", "Sneha", "Sonali", "Sonia", "Subhash", "Sudha", "Sudhir", "Sunil", "Sunita", "Suresh",
	"Sushma", "Swati", "Tanvi", "Tarun", "Uday", "Uma", "Varun", "Venkat", "Vijay", "Vikas", "Vikram", "Vinay",
	"Vinod", "Vishal", "Vivek", "Yash", "Yogesh", "Zoya", "Abhishek", "Aishwarya", "Alok", "Aman", "Amar",
	"Amrita", "Anita", "Apurva", "Arvind", "Asha", "Babita", "Balaji", "Bhaskar", "Chandra", "Chitra", "Darshan",
	"Devika", "Dhruv", "Ganesh", "Gayatri", "Hardik", "Harsha", "Hitesh", "Indira", "Ishaan", "Jagdish", "Jatin",
	"Jeevan", "Kalpana", "Kamal", "Kaveri", "Kirti", "Lavanya", "Leela", "Mallika", "Mangesh", "Manju", "Murali",
	"Nagaraj", "Nandini", "Narayan", "Nilesh", "Padma", "Parvati", "Pavan", "Pratap", "Preeti", "Puneet", "Radha",
	"Raghav", "Rajiv", "Ramya", "Ranjit", "Rashmi", "Reema", "Renu", "Rishi", "Sahil", "Sakshi", "Salman", "Samir",
	"Sanjana", "Satish", "Savita", "Shabana", "Sheetal", "Shiv", "Shobha", "Sita", "Soumya", "Srinivas", "Sumit",
	"Surya", "Tanya", "Tejas", "Trupti", "Usha", "Vaishali", "Vani", "Varsha", "Vidya", "Vimal", "Yamini", "Zara",
	"Aarti", "Gurpreet", "Harpreet", "Jaspreet", "Manpreet", "Simran", "Navjot", "Amandeep", "Kuldeep", "Baldev",
	"Ranbir", "Tapan", "Soumen", "Arnab", "Debashish", "Sourav", "Subrata", "Moumita", "Ananya", "Rituparna",
	"Lakshman", "Selvi", "Murugan", "Senthil", "Kavitha", "Revathi", "Saravanan", "Vignesh", "Yuvraj", "Abdul",
	"Ahmed", "Aslam", "Bilal", "Faisal", "Hassan", "Javed", "Nadeem", "Nasir", "Rizwan", "Sajid", "Shahid", "Tariq",
	"Yusuf", "Zaid", "Aamir", "Asif", "Danish", "Feroz", "Khalid", "Nazia", "Rubina", "Sana", "Shabnam", "Zeenat",
	"George", "Joseph", "Mathew", "Thomas", "Anna", "Mary", "Jacob", "Philip", "Sunny", "Alan", "Rita", "Nandita",
	"Bhavesh", "Jignesh", "Paresh", "Kalpesh", "Hiral", "Dhara", "Krupa", "Nirav", "Tushar", "Vipul", "Mitali",
	"Payal", "Isha", "Kriti", "Megha", "Nupur", "Pragya", "Richa", "Ruchi", "Shweta", "Tanuja", "Urmila", "Vandana",
}

var Surnames = []string{
	"Sharma", "Verma", "Gupta", "Singh", "Kumar", "Patel", "Shah", "Mehta", "Jain", "Agarwal", "Agrawal", "Bansal",
	"Mittal", "Goyal", "Garg", "Kapoor", "Khanna", "Malhotra", "Chopra", "Arora", "Bhatia", "Sethi", "Saxena",
	"Srivastava", "Mishra", "Tiwari", "Pandey", "Dubey", "Shukla", "Tripathi", "Chaturvedi", "Joshi", "Trivedi",
	"Desai", "Modi", "Parikh", "Bhatt", "Dave", "Pandya", "Rao", "Reddy", "Naidu", "Nair", "Menon", "Pillai", "Iyer",
	"Iyengar", "Krishnan", "Raman", "Subramaniam", "Venkatesan", "Murthy", "Shetty", "Hegde", "Kamath", "Pai", "Bhat",
	"Kulkarni", "Deshpande", "Patil", "Jadhav", "Pawar", "Shinde", "Gaikwad", "More", "Kadam", "Chavan", "Sawant",
	"Naik", "Gowda", "Das", "Dutta", "Ghosh", "Bose", "Banerjee", "Chatterjee", "Mukherjee", "Sen", "Roy",
	"Chakraborty", "Bhattacharya", "Majumdar", "Sarkar", "Mondal", "Saha", "Dey", "Khan", "Ansari", "Qureshi",
	"Siddiqui", "Sheikh", "Shaikh", "Pathan", "Syed", "Mirza", "Hussain", "Ali", "Rahman", "Malik", "Chauhan",
	"Rajput", "Rathore", "Yadav", "Choudhary", "Chaudhary", "Bishnoi", "Gill", "Sandhu", "Dhillon", "Sidhu",
	"Grewal", "Brar", "Bajwa", "Kaur", "Bedi", "Thakur", "Rana", "Negi", "Rawat", "Bisht", "Pant", "Bhandari",
	"Prasad", "Mahto", "Mandal", "Soren", "Nayak", "Behera", "Mohanty", "Sahoo", "Swain", "Pradhan", "Panda",
	"Fernandes", "D'Souza", "Pinto", "Lobo", "Gomes", "Rodrigues", "Pereira", "Varghese", "Kurian", "Chacko",
	"Abraham", "Sebastian", "Ramaswamy", "Natarajan", "Sundaram", "Balasubramanian", "Chandrasekhar", "Srinivasan",
	"Gopalan", "Raghavan", "Ramachandran", "Lakshmanan", "Venkataraman", "Sastry", "Acharya", "Bhagat", "Solanki",
	"Chavda", "Vaghela", "Jadeja", "Makwana", "Prajapati", "Suthar", "Rathod", "Parmar", "Zala", "Lal", "Devi",
	"Kumari", "Wadhwa", "Tandon", "Nanda", "Sood", "Luthra", "Puri", "Dhawan", "Vohra", "Ahuja", "Kohli", "Dua",
	"Oberoi", "Sahni", "Mehra", "Bhalla", "Chadha", "Kalra", "Talwar", "Suri", "Madan", "Sinha", "Jha", "Ojha",
	"Upadhyay", "Dwivedi", "Awasthi", "Bajpai", "Nigam", "Rastogi", "Khare", "Dixit", "Pathak", "Vyas", "Purohit",
	"Hazarika", "Baruah", "Gogoi", "Bora", "Deka", "Kalita", "Lama", "Tamang", "Rai", "Limbu", "Gurung",
}

var Titles = []string{"Mr", "Mrs", "Ms", "Miss", "Shri", "Smt", "Sri", "Dr", "Prof", "Er", "Adv", "Kumari", "Mx", "Md", "Mohd", "Sardar", "Late"}

var Cities = []string{
	"Mumbai", "Delhi", "New Delhi", "Bengaluru", "Bangalore", "Hyderabad", "Chennai", "Kolkata", "Pune", "Ahmedabad",
	"Jaipur", "Surat", "Lucknow", "Kanpur", "Nagpur", "Indore", "Thane", "Bhopal", "Visakhapatnam", "Patna",
	"Vadodara", "Ghaziabad", "Ludhiana", "Agra", "Nashik", "Faridabad", "Meerut", "Rajkot", "Varanasi", "Srinagar",
	"Aurangabad", "Dhanbad", "Amritsar", "Navi Mumbai", "Prayagraj", "Ranchi", "Howrah", "Coimbatore", "Jabalpur",
	"Gwalior", "Vijayawada", "Jodhpur", "Madurai", "Raipur", "Kota", "Guwahati", "Chandigarh", "Solapur", "Hubli",
	"Mysuru", "Mysore", "Tiruchirappalli", "Trichy", "Bareilly", "Aligarh", "Tiruppur", "Gurugram", "Gurgaon",
	"Noida", "Moradabad", "Jalandhar", "Bhubaneswar", "Salem", "Warangal", "Guntur", "Saharanpur", "Gorakhpur",
	"Bikaner", "Amravati", "Jamshedpur", "Bhilai", "Cuttack", "Kochi", "Ernakulam", "Thiruvananthapuram",
	"Trivandrum", "Kozhikode", "Thrissur", "Dehradun", "Mangaluru", "Mangalore", "Udaipur", "Ajmer", "Bokaro",
	"Durgapur", "Siliguri", "Asansol", "Nellore", "Tirupati", "Kurnool", "Rourkela", "Belagavi", "Belgaum", "Jammu",
	"Shimla", "Panaji", "Margao", "Puducherry", "Pondicherry", "Vellore", "Erode", "Kolhapur", "Sangli", "Satara",
	"Latur", "Nanded", "Akola", "Ujjain", "Dewas", "Rewa", "Jhansi", "Mathura", "Haridwar", "Rishikesh", "Ambala",
	"Panipat", "Karnal", "Rohtak", "Hisar", "Patiala", "Bathinda", "Mohali", "Zirakpur", "Hosur", "Tumakuru",
	"Shivamogga", "Davangere", "Ballari", "Vizag", "Secunderabad", "Kakinada", "Rajahmundry", "Imphal", "Shillong",
	"Agartala", "Aizawl", "Kohima", "Itanagar", "Gangtok", "Dimapur", "Greater Noida", "Pimpri Chinchwad",
}

var States = []string{
	"Andhra Pradesh", "Arunachal Pradesh", "Assam", "Bihar", "Chhattisgarh", "Goa", "Gujarat", "Haryana",
	"Himachal Pradesh", "Jharkhand", "Karnataka", "Kerala", "Madhya Pradesh", "Maharashtra", "Manipur", "Meghalaya",
	"Mizoram", "Nagaland", "Odisha", "Punjab", "Rajasthan", "Sikkim", "Tamil Nadu", "Telangana", "Tripura",
	"Uttar Pradesh", "Uttarakhand", "West Bengal", "Delhi", "Jammu and Kashmir", "Ladakh", "Puducherry", "Chandigarh",
}

// Localities are neighbourhood names used in synthetic addresses.
var Localities = []string{
	"Andheri East", "Andheri West", "Bandra West", "Borivali", "Dadar", "Powai", "Malad West", "Kandivali", "Goregaon",
	"Juhu", "Worli", "Colaba", "Chembur", "Mulund", "Vashi", "Kharghar", "Koramangala", "Indiranagar", "Whitefield",
	"Marathahalli", "HSR Layout", "Jayanagar", "JP Nagar", "Electronic City", "Hebbal", "Yelahanka", "BTM Layout",
	"Banashankari", "Malleshwaram", "Rajajinagar", "Madhapur", "Gachibowli", "Kukatpally", "Banjara Hills",
	"Jubilee Hills", "Ameerpet", "Begumpet", "Kondapur", "Hitech City", "Miyapur", "Adyar", "Anna Nagar", "T Nagar",
	"Velachery", "Tambaram", "Porur", "Guindy", "Mylapore", "Salt Lake", "New Town", "Park Street", "Ballygunge",
	"Behala", "Dum Dum", "Garia", "Kothrud", "Hinjewadi", "Wakad", "Baner", "Aundh", "Viman Nagar", "Kharadi",
	"Hadapsar", "Shivajinagar", "Pimpri", "Chinchwad", "Satellite", "Navrangpura", "Maninagar", "Bopal", "Vastrapur",
	"Prahlad Nagar", "Malviya Nagar", "Vaishali Nagar", "Mansarovar", "C Scheme", "Lajpat Nagar", "Saket", "Dwarka",
	"Rohini", "Pitampura", "Janakpuri", "Karol Bagh", "Connaught Place", "Vasant Kunj", "Greater Kailash",
	"Hauz Khas", "Mayur Vihar", "Laxmi Nagar", "Preet Vihar", "Shahdara", "Gomti Nagar", "Hazratganj", "Aliganj",
	"Indira Nagar", "Alambagh", "Boring Road", "Kankarbagh", "Civil Lines", "Model Town", "Sector 15", "Sector 62",
	"Sector 44", "Phase 2", "Rajendra Nagar", "Gandhi Nagar", "Nehru Nagar", "Shastri Nagar", "Ashok Nagar",
	"Ambedkar Nagar", "Kalyan Nagar", "Ram Nagar", "Krishna Nagar", "Tilak Nagar", "Patel Nagar", "Subhash Nagar",
}

var StreetWords = []string{
	"Road", "Rd", "Street", "St", "Main Road", "Cross", "Main", "Lane", "Marg", "Path", "Gali", "Chowk", "Circle",
	"Layout", "Extension", "Extn", "Enclave", "Colony", "Nagar", "Vihar", "Puram", "Bazar", "Bazaar", "Market",
	"Industrial Area", "Industrial Estate", "MIDC", "Tech Park", "IT Park",
}

var BuildingWords = []string{
	"Apartments", "Apartment", "Apts", "Residency", "Towers", "Tower", "Heights", "Society", "CHS", "Complex",
	"Plaza", "Arcade", "Building", "Bldg", "Villa", "Bungalow", "Niwas", "Sadan", "Bhavan", "Kunj", "Mansion",
	"Palace", "Court", "Garden", "Greens", "Enclave", "Estate", "Park", "Nest", "Homes", "Meadows",
}

// AddressCues are lowercase words whose presence near a 6-digit number
// suggests an address.
var AddressCues = func() map[string]bool {
	m := map[string]bool{}
	add := func(list []string) {
		for _, s := range list {
			for _, w := range strings.Fields(strings.ToLower(s)) {
				if len(w) > 1 {
					m[w] = true
				}
			}
		}
	}
	add(StreetWords)
	add(BuildingWords)
	add(Cities)
	add(States)
	add(Localities)
	for _, w := range []string{"near", "opp", "opposite", "behind", "beside", "next", "adjacent", "landmark",
		"village", "post", "po", "taluk", "taluka", "tehsil", "mandal", "district", "dist", "dt", "pin", "pincode",
		"flat", "floor", "wing", "plot", "house", "door", "block", "sector", "phase", "gate", "station", "junction",
		"temple", "school", "college", "hospital", "mall", "lake", "hills", "cantt", "cantonment", "camp", "sez",
		"india", "no", "h", "d", "b", "c", "a"} {
		m[w] = true
	}
	return m
}()

// Stop words that look like names when capitalised but never are.
var NameStop = toSet([]string{
	"bank", "account", "limited", "pvt", "ltd", "private", "please", "dear", "thank", "thanks", "regards", "sir",
	"madam", "team", "hello", "hi", "note", "kindly", "customer", "care", "support", "service", "services", "india",
	"the", "and", "for", "from", "with", "your", "our", "this", "that", "yes", "no", "not", "available", "na", "nil",
	"null", "none", "test", "user", "admin", "sample", "demo", "dummy", "unknown", "name", "number", "card",
	"aadhaar", "aadhar", "pan", "gst", "gstin", "kyc", "otp", "upi", "ifsc", "branch", "savings", "current",
	"credit", "debit", "loan", "emi", "invoice", "order", "payment", "transaction", "txn", "ref", "reference",
	"status", "pending", "approved", "rejected", "failed", "success", "error", "warning", "info", "debug",
	"january", "february", "march", "april", "may", "june", "july", "august", "september", "october", "november",
	"december", "monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday", "state", "city",
	"district", "road", "street", "nagar", "colony", "sector", "block", "flat", "house", "plot", "near", "opp",
	"ltd.", "inc", "corp", "corporation", "company", "co", "group", "finance", "insurance", "mutual", "fund",
	"mobile", "phone", "email", "address", "date", "birth", "gender", "male", "female", "government", "govt",
	"income", "tax", "department", "unique", "identification", "authority", "virtual", "valid", "verified",
	"verify", "verification", "copy", "attached", "attachment", "document", "documents", "proof", "id",
	"mumbai", "delhi", "bengaluru", "bangalore", "hyderabad", "chennai", "kolkata", "pune", "ahmedabad", "jaipur",
	"amount", "total", "balance", "rupees", "only", "subject", "re", "fwd", "fw", "cc", "to", "hdfc", "icici",
	"sbi", "axis", "kotak", "paytm", "phonepe", "gpay", "amazon", "flipkart", "airtel", "jio", "vodafone", "bsnl",
	"father", "father's", "fathers", "mother", "mother's", "husband", "wife", "son", "daughter", "spouse", "guardian",
	"parent", "parents", "age", "sex", "gender", "assessment", "year", "class", "room", "warden", "agent", "officer",
	"manager", "director", "proprietor", "signatory", "holder", "nominee", "beneficiary", "applicant", "seva", "kendra",
	"online", "statements", "statement", "choosing", "please", "valid", "till", "blood", "group", "station", "police",
	"present", "permanent", "residential", "correspondence", "communication", "registered", "office", "home",
	"landmark", "village", "taluk", "tehsil", "post", "pin", "code", "number", "ending", "last", "digits", "linked",
	"seeded", "seeding", "verified", "verification", "pending", "ok", "yes", "none", "nil", "unknown", "self",
})

func toSet(words []string) map[string]bool {
	m := make(map[string]bool, len(words))
	for _, w := range words {
		m[strings.ToLower(w)] = true
	}
	return m
}

var (
	firstSet   = toSet(FirstNames)
	surnameSet = toSet(Surnames)
	titleSet   = toSet(Titles)
	citySet    = toSet(Cities)
	stateSet   = toSet(States)
)

func IsFirstName(w string) bool { return firstSet[strings.ToLower(strings.Trim(w, ".,"))] }
func IsSurname(w string) bool   { return surnameSet[strings.ToLower(strings.Trim(w, ".,"))] }
func IsTitle(w string) bool     { return titleSet[strings.ToLower(strings.Trim(w, "."))] }
func IsCity(w string) bool      { return citySet[strings.ToLower(w)] }
func IsState(w string) bool     { return stateSet[strings.ToLower(w)] }
func IsNameStop(w string) bool  { return NameStop[strings.ToLower(strings.Trim(w, ".,"))] }
func IsNameWord(w string) bool  { return IsFirstName(w) || IsSurname(w) }
