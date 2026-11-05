package synth

// Sentence templates. Placeholders: {L1}..{L3} label of field i, {V1}..{V3}
// value of field i (recorded as a truth span), {NAME} the persona's name
// (NAME span), {CITY}, {BANK}, {COMPANY}, {DATE} (a non-DOB date, "date"
// span), {D:kind} a distractor value of that kind (span). Templates are
// pooled into train-only / test-only / shared by hash (see Pool) so the test
// split contains unseen phrasings.

var prose1 = []string{
	"Please find my {L1} {V1} for KYC verification.",
	"My {L1} is {V1}.",
	"Hi, I want to update my {L1} to {V1}.",
	"Sharing {L1}: {V1} as requested.",
	"{L1} {V1} has been verified successfully.",
	"Kindly note the {L1} - {V1}.",
	"The customer's {L1} ({V1}) does not match our records.",
	"{V1} is my {L1}, please update.",
	"Attached is the copy of my {L1}. Number: {V1}.",
	"Dear team, {L1} {V1} needs correction.",
	"We received your request. {L1}: {V1}.",
	"As per the form, {L1} = {V1}.",
	"Confirming {L1} {V1} for {NAME}.",
	"{NAME} has submitted {L1} {V1}.",
	"Updated {L1} ({V1}) in CRM for {NAME}.",
	"Please verify {L1} {V1} belongs to {NAME}, {CITY}.",
	"{L1} provided by the applicant: {V1}",
	"Use {L1} {V1} for the application.",
	"Enter your {L1} ({V1}) on the portal.",
	"{L1} is {V1} as per the document submitted on {DATE}.",
	"Noted {L1} {V1}; will process by {DATE}.",
	"FYI - {L1} {V1}",
	"The {L1} on file is {V1}, is that correct?",
	"{L1}# {V1}",
	"{L1}:{V1}",
	"Record {L1} {V1} was flagged during audit.",
	"Could you confirm whether {V1} is the correct {L1}?",
	"We tried {L1} {V1} but the lookup failed.",
	"{NAME} from {CITY} shared {L1} {V1} over the call.",
	"Branch {BANK} received {L1} {V1}.",
	// several values under one label
	"{L1}: {V1} and {V1b}",
	"{L1}s of the couple: {V1} and {V1b}.",
	"{L1} {V1} / {V1b}",
	"{L1}: {V1}, {V1b}, {V1c}",
	"Both {L1} numbers ({V1} and {V1b}) are valid.",
	"Old {L1} {V1}, new {L1} {V1b}.",
	"{L1} {V1} replaced by {V1b}.",
	"{L1} list: {V1}; {V1b}; {V1c}",
	"{L1} - {V1}\n{L1} - {V1b}",
	"Family {L1}s: {NAME} {V1}, {NAME} {V1b}",
}

var prose2 = []string{
	"{L1} {V1}, {L2} {V2}",
	"{L1}: {V1}, {L2}: {V2}",
	"{L1} {V1} and {L2} {V2}",
	"Please link {L1} {V1} to {L2} {V2}.",
	"{L2} {V2} is linked with {L1} {V1}.",
	"Customer {NAME} ({L1} {V1}) has requested a change to {L2} {V2}.",
	"My {L1} is {V1} and my {L2} is {V2}.",
	"{L1}/{L2}: {V1}/{V2}",
	"{L1} - {V1} | {L2} - {V2}",
	"Kindly update {L1} {V1}; {L2} remains {V2}.",
	"{V1} ({L1}) and {V2} ({L2}) submitted.",
	"Verified {L1} {V1} against {L2} {V2}.",
	"{L1} {V1} {L2} {V2}",
	"Received {L1} {V1}. Also {L2} {V2} for reference.",
	"{NAME} - {L1} {V1}, {L2} {V2}, {CITY}",
	"For {L1} {V1} the {L2} on file is {V2}.",
	"Mismatch: {L1} {V1} vs {L2} {V2}.",
	"Please do not confuse {L1} {V1} with {L2} {V2}.",
	"{L1} {V1} was entered in the {L2} field instead of {V2}.",
	"Seeding of {L1} {V1} to {L2} {V2} completed.",
	"{L1} no {V1}, {L2} no {V2}",
	"{L1} {V1}\n{L2} {V2}",
	"{L1}={V1}&{L2}={V2}",
	"{L1} {V1} ; {L2} {V2}",
	"Updated {L1} to {V1}. Old {L2} {V2} retained.",
	"{NAME}: {L1} {V1} / {L2} {V2}",
	"The {L1} {V1} and the {L2} {V2} belong to the same customer.",
	"Txn declined for {L1} {V1} - {L2} {V2} blocked.",
	"{L2} {V2} ({L1} {V1})",
	"{L1} {V1} -> {L2} {V2}",
	"Both {L1} ({V1}) and {L2} ({V2}) are required.",
	"Sent OTP to {L2} {V2} for {L1} {V1}.",
	"{L1} {V1} debited, {L2} {V2}.",
	"ref {L1} {V1} {L2} {V2} status ok",
	"{L1}\t{V1}\t{L2}\t{V2}",
}

var prose3 = []string{
	"{L1} {V1}, {L2} {V2}, {L3} {V3}",
	"Details - {L1}: {V1}; {L2}: {V2}; {L3}: {V3}",
	"{NAME}, {L1} {V1}, {L2} {V2} and {L3} {V3}.",
	"Please update {L1} {V1}, {L2} {V2} and {L3} {V3} in the system.",
	"KYC received: {L1} {V1} | {L2} {V2} | {L3} {V3}",
	"{L1} {V1}. {L2} {V2}. {L3} {V3}.",
	"{L1} {V1} {L2} {V2} {L3} {V3}",
	"Txn for {NAME}: {L1} {V1}, {L2} {V2}, {L3} {V3} on {DATE}",
	"{L1}={V1} {L2}={V2} {L3}={V3}",
	"{L1}: {V1}\n{L2}: {V2}\n{L3}: {V3}",
	"Applicant {NAME} ({CITY}) - {L1} {V1}; {L2} {V2}; {L3} {V3}",
	"{L1} {V1}/{L2} {V2}/{L3} {V3}",
	"Captured {L1} ({V1}), {L2} ({V2}), {L3} ({V3}).",
	"{L1} {V1}, and {L2} {V2}, then {L3} {V3}.",
	"{L1} - {V1}, {L2} - {V2}, {L3} - {V3}",
}

// keywordOnly sentences mention labels without a value; they create
// keyword windows around neighbouring distractors.
var keywordOnly = []string{
	"Please share your {L1} copy for verification.",
	"{L1} seeding is pending.",
	"We could not verify the {L1} provided.",
	"Do not share your {L1} with anyone.",
	"{L1} is mandatory for this service.",
	"Upload {L1} and {L2} documents.",
	"Your {L1} has been linked successfully.",
	"Reminder: {L1} verification due by {DATE}.",
	"{L1} update failed with error code {D:generic_id}.",
	"{L1} status: pending. Ticket {D:ticket}.",
	"Request for {L1} change received on {DATE}, ref {D:rrn}.",
	"{L1} document rejected - image blurry. Order {D:order_id}.",
	"Carry original {L1} and {L2} to the branch.",
	"{L1} not available. Customer ID {D:customer_id}.",
	"Kindly confirm {L1} over the registered contact.",
	"{L1} masked as per policy; txn {D:txn_id} completed.",
	"The {L1} field was left blank in application {D:customer_id}.",
	"{L1} verification charges {D:amount} apply.",
	"{L1} copy attached (scanned {D:count} pages).",
	"Total {D:count} {L1} records processed in batch {D:ticket}.",
	"{L1} / {L2} proof required.",
	"{L1} ekyc OTP {D:otp} sent.",
}

var filler = []string{
	"Thanks for reaching out to customer care.",
	"Please allow 2-3 working days for processing.",
	"This is a system generated message, do not reply.",
	"Regards, Support Team",
	"Let me know if you need anything else.",
	"Your request has been forwarded to the branch.",
	"Amount of {D:amount} will be credited within 48 hours.",
	"Order {D:order_id} shipped via BlueDart, AWB {D:tracking}.",
	"Payment of {D:amount} received, UPI Ref {D:rrn}.",
	"Processed {D:count} records in {D:count} ms.",
	"Meeting scheduled on {DATE} at 3 PM.",
	"Ticket {D:ticket} has been resolved.",
	"Server responded with status 200 in 120ms.",
	"Invoice {D:invoice} for {D:amount} is overdue.",
	"Hope you are doing well.",
	"Please find the details below.",
	"As discussed on the call today.",
	"Note: charges are inclusive of GST.",
	"Happy to help!",
	"Visit the nearest branch with original documents.",
	"Your policy {D:policy} is due for renewal on {DATE}.",
	"Consumer {D:consumer_no} bill of {D:amount} generated.",
	"Device {D:device_id} registered on {DATE}.",
	"PNR {D:pnr} confirmed, seat 42A.",
	"Cheque {D:cheque} cleared on {DATE}.",
	"Balance as on {DATE}: {D:amount}",
	"Last login from {CITY} on {DATE}.",
	"Thank you for banking with {BANK}.",
	"Interest rate {D:percent} p.a. applicable.",
	"Vehicle {D:vehicle} challan {D:invoice} paid.",
	"Your OTP is {D:otp}. Valid for 10 minutes.",
	"Welcome to {COMPANY}.",
	"Please quote ticket {D:ticket} in future correspondence.",
	"Transaction {D:txn_id} of {D:amount} failed; amount will be reversed.",
	"EMI of {D:amount} debited on {DATE}.",
}

// hardLayouts2/3 separate labels from their values the way tables, aligned
// forms and slash-separated pairs do; the nearest-left attribution rule is
// only meaningful if the corpus exercises these.
var hardLayouts2 = []string{
	"{L1} / {L2} : {V1} / {V2}",
	"{L1}, {L2}: {V1}, {V2}",
	"{L1}\t{L2}\n{V1}\t{V2}",
	"{L1} | {L2}\n{V1} | {V2}",
	"{L1}    {L2}\n{V1}    {V2}",
	"{V1} - {L1}; {V2} - {L2}",
	"{L1} and {L2} are {V1} and {V2} respectively.",
	"({L1}, {L2}) = ({V1}, {V2})",
	"{L1}={V1}&{L2}={V2}",
	"{V1}, {V2} ({L1}, {L2})",
}

var hardLayouts3 = []string{
	"{L1} / {L2} / {L3} : {V1} / {V2} / {V3}",
	"{L1}\t{L2}\t{L3}\n{V1}\t{V2}\t{V3}",
	"{L1} | {L2} | {L3}\n{V1} | {V2} | {V3}",
	"{V1}, {V2}, {V3} ({L1}, {L2}, {L3})",
	"{L1}, {L2} and {L3}: {V1}, {V2}, {V3}",
	"{L1}   {L2}   {L3}\n{V1}   {V2}   {V3}",
}

// mobileCtx are label-free phone contexts ("call me on ...").
var mobileCtx = []string{
	"Call me on {V1} after 6 pm.",
	"Reach me at {V1} for any clarification.",
	"Customer called from {V1} regarding the refund.",
	"OTP sent to {V1}.",
	"SMS delivered to {V1} at 10:42.",
	"You can WhatsApp me on {V1}.",
	"Missed call from {V1}, please call back.",
	"Dial {V1} to speak to {NAME}.",
	"{NAME} ({V1}) will collect the parcel.",
	"Text me on {V1} once done.",
}

var sectionHeaders = []string{
	"KYC DETAILS", "Bank Details", "Personal Information", "=== Customer Profile ===", "Applicant Details",
	"--- Account Information ---", "CUSTOMER DETAILS", "Nominee Details", "Contact Information", "Payment Details",
	"Verification Summary", "Form 60 / KYC", "ONBOARDING FORM", "Loan Application", "Employee Record",
	"Policy Holder Details", "Shipping Information", "# Customer", "** KYC **", "Identity Proof",
}

var services = []string{
	"kyc-service", "payments-api", "onboarding", "auth-svc", "notification-svc", "loan-engine", "upi-gateway",
	"crm-sync", "card-issuer", "statement-gen", "ekyc-worker", "ledger", "risk-engine", "support-portal", "batch-job",
}

var logLevels = []string{"INFO", "INFO", "INFO", "DEBUG", "WARN", "ERROR"}

var logMessages = []string{
	"kyc verify {K1}={V1} {K2}={V2} status=OK",
	"request received {K1}={V1}",
	"linking {K1}={V1} to {K2}={V2}",
	"validation failed for {K1}={V1} reason=checksum",
	"{K1}={V1} {K2}={V2} {K3}={V3} latency=42ms",
	"user lookup by {K1} {V1} returned 1 row",
	"processing {K1}: {V1}, {K2}: {V2}",
	"masked {K1}={V1} before persisting",
	"{K1}={V1} not found; fallback to {K2}={V2}",
	"audit {K1}={V1} action=UPDATE actor=system",
	"payment settled {K1}={V1} {K2}={V2} amount={D:amount}",
	"retry 3/5 for {K1}={V1} err=timeout",
	"cache miss key={K1}:{V1}",
	"{K1} {V1} | {K2} {V2} | ok",
	"GET /v1/customers?{K1}={V1}&{K2}={V2} 200",
	"POST /kyc {\"{K1}\":\"{V1}\",\"{K2}\":\"{V2}\"} -> 201",
	"job done rows={D:count} {K1}={V1}",
	"otp sent {K1}={V1} ttl=300",
	"dedupe hit {K1}={V1} existing {K2}={V2}",
	"{K1}={V1} {K2}={V2}",
	"login ok uid={D:user_id} ip=10.{D:count}.1.7 {K1}={V1}",
	"session created for uid {D:user_id}; {K1}={V1}",
	"user uid={D:user_id} not found while linking {K1}={V1}",
	"actor uid={D:user_id} updated {K1} {V1} -> {K2} {V2}",
}

var emailOpeners = []string{"Hi,", "Hello team,", "Dear Sir/Madam,", "Hi Support,", "Dear {NAME},", "Hello,", "Respected Sir,", "Team,", "Hi {NAME},"}
var emailClosers = []string{"Thanks,\n{NAME}", "Regards,\n{NAME}", "Thanks & Regards,\n{NAME}\n{CITY}", "Best,\n{NAME}", "Warm regards\n{NAME}", "Thank you.", "Regards,\nSupport Team"}
var subjects = []string{"Subject: KYC update", "Subject: Re: Account linking", "Subject: Fwd: Documents", "Subject: Aadhaar seeding request", "Subject: Payment issue", "Subject: Address change", "Subject: Re: Re: Verification pending", "Subject: Loan application {D:customer_id}", "Subject: Ticket {D:ticket}", "Subject: Refund for order {D:order_id}"}
