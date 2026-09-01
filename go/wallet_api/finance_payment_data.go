package main

// finance_payment_data.go — the P2P marketplace payment-method catalog:
// 881 local payment methods across all 238 countries.
//
// Composition (exact counts enforced by TestFinancePaymentCatalog):
//   - 3 generic rails per country (domestic bank transfer, mobile money,
//     instant payment) = 3 x 238 = 714;
//   - 167 curated named methods (SEPA, UPI, Pix, M-Pesa, bKash, GCash,
//     Interac, Zelle, ...) with their real country coverage.
//
// This is catalog/registry data (like the chain registry), not mock data:
// every named method is a real payment rail.

import "strings"

type paymentMethod struct {
	Code      string   `json:"code"`
	Name      string   `json:"name"`
	Kind      string   `json:"kind"` // bank | mobile
	Countries []string `json:"countries,omitempty"` // empty = all 238
}

// financeCountryCodes — ISO 3166-1 alpha-2, 238 codes.
var financeCountryCodes = []string{
	"AF", "AL", "DZ", "AS", "AD", "AO", "AI", "AG", "AR", "AM",
	"AW", "AU", "AT", "AZ", "BS", "BH", "BD", "BB", "BY", "BE",
	"BZ", "BJ", "BM", "BT", "BO", "BA", "BW", "BR", "BN", "BG",
	"BF", "BI", "CV", "KH", "CM", "CA", "KY", "CF", "TD", "CL",
	"CN", "CO", "KM", "CG", "CD", "CK", "CR", "CI", "HR", "CU",
	"CY", "CZ", "DK", "DJ", "DM", "DO", "EC", "EG", "SV", "GQ",
	"ER", "EE", "SZ", "ET", "FK", "FO", "FJ", "FI", "FR", "GF",
	"PF", "GA", "GM", "GE", "DE", "GH", "GI", "GR", "GL", "GD",
	"GP", "GU", "GT", "GG", "GN", "GW", "GY", "HT", "VA", "HN",
	"HK", "HU", "IS", "IN", "ID", "IR", "IQ", "IE", "IM", "IL",
	"IT", "JM", "JP", "JE", "JO", "KZ", "KE", "KI", "KP", "KR",
	"KW", "KG", "LA", "LV", "LB", "LS", "LR", "LY", "LI", "LT",
	"LU", "MO", "MG", "MW", "MY", "MV", "ML", "MT", "MH", "MQ",
	"MR", "MU", "YT", "MX", "FM", "MD", "MC", "MN", "ME", "MS",
	"MA", "MZ", "MM", "NA", "NR", "NP", "NL", "NC", "NZ", "NI",
	"NE", "NG", "NU", "NF", "MK", "MP", "NO", "OM", "PK", "PW",
	"PS", "PA", "PG", "PY", "PE", "PH", "PN", "PL", "PT", "PR",
	"QA", "RE", "RO", "RU", "RW", "BL", "SH", "KN", "LC", "MF",
	"PM", "VC", "WS", "SM", "ST", "SA", "SN", "RS", "SC", "SL",
	"SG", "SX", "SK", "SI", "SB", "SO", "ZA", "SS", "ES", "LK",
	"SD", "SR", "SJ", "SE", "CH", "SY", "TW", "TJ", "TZ", "TH",
	"TL", "TG", "TK", "TO", "TT", "TN", "TR", "TM", "TC", "TV",
	"UG", "UA", "AE", "GB", "US", "UY", "UZ", "VU", "VE", "VN",
	"VG", "VI", "WF", "EH", "YE", "ZM", "ZW", "CW",
}

// eeaCountries — SEPA zone.
var eeaCountries = []string{
	"AT", "BE", "BG", "HR", "CY", "CZ", "DK", "EE", "FI", "FR",
	"DE", "GR", "HU", "IS", "IE", "IT", "LV", "LI", "LT", "LU",
	"MT", "NL", "NO", "PL", "PT", "RO", "SK", "SI", "ES", "SE",
}

// curatedPaymentMethods — real named rails with real coverage.
var curatedPaymentMethods = []paymentMethod{
	// ---- Global ----
	{"swift", "SWIFT Bank Transfer", "bank", nil},
	{"wire_transfer", "International Wire Transfer", "bank", nil},
	{"western_union", "Western Union", "bank", nil},
	{"moneygram", "MoneyGram", "bank", nil},
	{"wise", "Wise", "bank", nil},
	{"payoneer", "Payoneer", "bank", nil},
	{"paypal", "PayPal", "mobile", nil},
	{"revolut", "Revolut", "mobile", nil},
	{"skrill", "Skrill", "mobile", nil},
	{"neteller", "NETELLER", "mobile", nil},
	{"apple_pay", "Apple Pay", "mobile", nil},
	{"google_pay", "Google Pay", "mobile", nil},
	// ---- North America ----
	{"ach", "ACH Transfer", "bank", []string{"US"}},
	{"fedwire", "Fedwire", "bank", []string{"US"}},
	{"zelle", "Zelle", "mobile", []string{"US"}},
	{"venmo", "Venmo", "mobile", []string{"US"}},
	{"cashapp", "Cash App", "mobile", []string{"US"}},
	{"interac", "Interac e-Transfer", "mobile", []string{"CA"}},
	// ---- Europe ----
	{"sepa", "SEPA Credit Transfer", "bank", eeaCountries},
	{"sepa_instant", "SEPA Instant", "bank", eeaCountries},
	{"ideal", "iDEAL", "bank", []string{"NL"}},
	{"giropay", "giropay", "bank", []string{"DE"}},
	{"sofort", "Sofort (Klarna)", "bank", []string{"DE", "AT", "CH", "BE"}},
	{"eps", "EPS Uberweisung", "bank", []string{"AT"}},
	{"przelewy24", "Przelewy24", "bank", []string{"PL"}},
	{"blik", "BLIK", "mobile", []string{"PL"}},
	{"multibanco", "Multibanco", "bank", []string{"PT"}},
	{"mbway", "MB Way", "mobile", []string{"PT"}},
	{"bizum", "Bizum", "mobile", []string{"ES"}},
	{"swish", "Swish", "mobile", []string{"SE"}},
	{"mobilepay", "MobilePay", "mobile", []string{"DK", "FI"}},
	{"vipps", "Vipps", "mobile", []string{"NO"}},
	{"trustly", "Trustly", "bank", []string{"SE", "FI", "DK", "NO", "DE", "NL", "GB", "ES", "IT", "FR", "PL"}},
	{"klarna", "Klarna", "bank", []string{"DE", "SE", "NO", "FI", "DK", "NL", "AT", "CH", "GB", "US"}},
	{"bancontact", "Bancontact", "bank", []string{"BE"}},
	{"fps", "Faster Payments", "bank", []string{"GB"}},
	{"bacs", "BACS", "bank", []string{"GB"}},
	{"chaps", "CHAPS", "bank", []string{"GB"}},
	{"monzo", "Monzo", "mobile", []string{"GB"}},
	{"satispay", "Satispay", "mobile", []string{"IT"}},
	{"lydia", "Lydia", "mobile", []string{"FR"}},
	{"carte_bancaire", "Carte Bancaire", "bank", []string{"FR"}},
	{"twint", "TWINT", "mobile", []string{"CH"}},
	{"sbp", "Faster Payments System (SBP)", "bank", []string{"RU"}},
	{"tinkoff_pay", "Tinkoff Pay", "mobile", []string{"RU"}},
	{"sberpay", "SberPay", "mobile", []string{"RU"}},
	{"yoomoney", "YooMoney", "mobile", []string{"RU"}},
	{"qiwi", "QIWI", "mobile", []string{"RU"}},
	{"payu", "PayU", "bank", []string{"PL", "CZ", "RO", "HU"}},
	// ---- Asia ----
	{"upi", "UPI", "mobile", []string{"IN"}},
	{"paytm", "Paytm", "mobile", []string{"IN"}},
	{"phonepe", "PhonePe", "mobile", []string{"IN"}},
	{"bhim", "BHIM", "mobile", []string{"IN"}},
	{"imps", "IMPS", "bank", []string{"IN"}},
	{"neft", "NEFT", "bank", []string{"IN"}},
	{"rtgs", "RTGS", "bank", []string{"IN"}},
	{"bkash", "bKash", "mobile", []string{"BD"}},
	{"nagad", "Nagad", "mobile", []string{"BD"}},
	{"rocket", "Rocket", "mobile", []string{"BD"}},
	{"upay_bd", "Upay", "mobile", []string{"BD"}},
	{"jazzcash", "JazzCash", "mobile", []string{"PK"}},
	{"easypaisa", "Easypaisa", "mobile", []string{"PK"}},
	{"nayapay", "NayaPay", "mobile", []string{"PK"}},
	{"sadapay", "SadaPay", "mobile", []string{"PK"}},
	{"esewa", "eSewa", "mobile", []string{"NP"}},
	{"khalti", "Khalti", "mobile", []string{"NP"}},
	{"fonepay", "FonePay", "mobile", []string{"NP"}},
	{"imepay", "IME Pay", "mobile", []string{"NP"}},
	{"alipay", "Alipay", "mobile", []string{"CN"}},
	{"wechat_pay", "WeChat Pay", "mobile", []string{"CN"}},
	{"unionpay", "UnionPay", "bank", []string{"CN"}},
	{"kakaopay", "KakaoPay", "mobile", []string{"KR"}},
	{"naverpay", "Naver Pay", "mobile", []string{"KR"}},
	{"toss", "Toss", "mobile", []string{"KR"}},
	{"paypay", "PayPay", "mobile", []string{"JP"}},
	{"linepay", "LINE Pay", "mobile", []string{"JP"}},
	{"rakutenpay", "Rakuten Pay", "mobile", []string{"JP"}},
	{"promptpay", "PromptPay", "mobile", []string{"TH"}},
	{"truemoney", "TrueMoney", "mobile", []string{"TH"}},
	{"shopeepay", "ShopeePay", "mobile", []string{"TH", "ID", "VN", "MY", "PH", "SG"}},
	{"rabbit_line_pay", "Rabbit LINE Pay", "mobile", []string{"TH"}},
	{"duitnow", "DuitNow", "mobile", []string{"MY"}},
	{"tng_ewallet", "Touch 'n Go eWallet", "mobile", []string{"MY"}},
	{"boost", "Boost", "mobile", []string{"MY"}},
	{"grabpay", "GrabPay", "mobile", []string{"MY", "SG", "PH", "ID", "VN", "TH"}},
	{"gcash", "GCash", "mobile", []string{"PH"}},
	{"maya", "Maya", "mobile", []string{"PH"}},
	{"coinsph", "Coins.ph", "mobile", []string{"PH"}},
	{"ovo", "OVO", "mobile", []string{"ID"}},
	{"dana", "DANA", "mobile", []string{"ID"}},
	{"gopay", "GoPay", "mobile", []string{"ID"}},
	{"linkaja", "LinkAja", "mobile", []string{"ID"}},
	{"zalopay", "ZaloPay", "mobile", []string{"VN"}},
	{"momo_vn", "MoMo", "mobile", []string{"VN"}},
	{"vnpay", "VNPay", "mobile", []string{"VN"}},
	{"vietqr", "VietQR", "bank", []string{"VN"}},
	{"wing", "Wing Money", "mobile", []string{"KH"}},
	{"pipay", "Pi Pay", "mobile", []string{"KH"}},
	{"bakong", "Bakong", "mobile", []string{"KH"}},
	{"kbzpay", "KBZPay", "mobile", []string{"MM"}},
	{"wavepay", "Wave Money", "mobile", []string{"MM"}},
	// ---- Middle East / Central Asia ----
	{"stcpay", "STC Pay", "mobile", []string{"SA"}},
	{"mada_pay", "mada Pay", "mobile", []string{"SA"}},
	{"urpay", "urpay", "mobile", []string{"SA"}},
	{"benefitpay", "BenefitPay", "mobile", []string{"BH"}},
	{"fawry", "Fawry", "mobile", []string{"EG"}},
	{"instapay", "InstaPay", "mobile", []string{"EG"}},
	{"telda", "Telda", "mobile", []string{"EG"}},
	{"vodafone_cash_eg", "Vodafone Cash Egypt", "mobile", []string{"EG"}},
	{"orange_cash_eg", "Orange Cash Egypt", "mobile", []string{"EG"}},
	{"etisalat_cash", "Etisalat Cash", "mobile", []string{"EG"}},
	{"zaincash", "Zain Cash", "mobile", []string{"IQ"}},
	{"asiahawala", "Asia Hawala", "mobile", []string{"IQ"}},
	{"fastpay_iq", "FastPay Iraq", "mobile", []string{"IQ"}},
	{"cliq", "CliQ", "mobile", []string{"JO"}},
	{"kaspikz", "Kaspi.kz", "mobile", []string{"KZ"}},
	{"halyk", "Halyk Bank", "bank", []string{"KZ"}},
	{"payme_uz", "Payme", "mobile", []string{"UZ"}},
	{"click_uz", "Click", "mobile", []string{"UZ"}},
	{"humo", "Humo", "bank", []string{"UZ"}},
	{"uzum", "Uzum Bank", "mobile", []string{"UZ"}},
	// ---- Africa ----
	{"mpesa", "M-Pesa", "mobile", []string{"KE", "TZ"}},
	{"mpesa_vodacom", "Vodacom M-Pesa", "mobile", []string{"TZ", "MZ", "CD", "LS"}},
	{"mtn_momo", "MTN Mobile Money", "mobile", []string{"GH", "UG", "RW", "CM", "CI", "ZM"}},
	{"airtel_money", "Airtel Money", "mobile", []string{"KE", "UG", "TZ", "RW", "ZM", "MW", "NG"}},
	{"orange_money", "Orange Money", "mobile", []string{"SN", "CI", "ML", "BF", "CM"}},
	{"tigo_money", "Tigo Money", "mobile", []string{"GH", "TZ", "RW"}},
	{"ecocash", "EcoCash", "mobile", []string{"ZW"}},
	{"onemoney", "OneMoney", "mobile", []string{"ZW"}},
	{"telebirr", "Telebirr", "mobile", []string{"ET"}},
	{"amole", "Amole", "mobile", []string{"ET"}},
	{"cbe_birr", "CBE Birr", "mobile", []string{"ET"}},
	{"wave", "Wave", "mobile", []string{"SN", "CI", "UG"}},
	{"freemoney", "Free Money", "mobile", []string{"SN"}},
	{"moov_money", "Moov Money", "mobile", []string{"CI", "BJ", "TG", "BF", "ML", "NE"}},
	{"tmoney", "T-Money", "mobile", []string{"TG"}},
	{"paga", "Paga", "mobile", []string{"NG"}},
	{"opay", "OPay", "mobile", []string{"NG"}},
	{"palmpay", "PalmPay", "mobile", []string{"NG", "GH"}},
	{"kuda", "Kuda", "mobile", []string{"NG"}},
	{"moniepoint", "Moniepoint", "mobile", []string{"NG"}},
	{"chipper", "Chipper Cash", "mobile", []string{"NG", "GH", "KE", "UG", "ZA", "RW"}},
	{"vodafone_cash_gh", "Vodafone Cash Ghana", "mobile", []string{"GH"}},
	// ---- Latin America ----
	{"pix", "Pix", "mobile", []string{"BR"}},
	{"boleto", "Boleto Bancario", "bank", []string{"BR"}},
	{"picpay", "PicPay", "mobile", []string{"BR"}},
	{"mercadopago", "Mercado Pago", "mobile", []string{"BR", "AR", "MX", "CL", "CO", "UY", "PE"}},
	{"spei", "SPEI", "bank", []string{"MX"}},
	{"oxxo", "OXXO Pay", "bank", []string{"MX"}},
	{"codi", "CoDi", "mobile", []string{"MX"}},
	{"pse", "PSE", "bank", []string{"CO"}},
	{"nequi", "Nequi", "mobile", []string{"CO"}},
	{"daviplata", "Daviplata", "mobile", []string{"CO"}},
	{"yape", "Yape", "mobile", []string{"PE"}},
	{"plin", "Plin", "mobile", []string{"PE"}},
	{"pagoefectivo", "PagoEfectivo", "bank", []string{"PE"}},
	{"webpay", "Webpay", "bank", []string{"CL"}},
	{"mach", "MACH", "mobile", []string{"CL"}},
	{"cuentarut", "CuentaRUT", "bank", []string{"CL"}},
	{"khipu", "Khipu", "bank", []string{"CL"}},
	{"servipag", "Servipag", "bank", []string{"CL"}},
	{"uala", "Uala", "mobile", []string{"AR"}},
	{"brubank", "Brubank", "mobile", []string{"AR"}},
	{"rapipago", "Rapipago", "bank", []string{"AR"}},
	{"pagofacil", "Pago Facil", "bank", []string{"AR"}},
	// ---- Oceania ----
	{"payid", "PayID", "mobile", []string{"AU"}},
	{"osko", "Osko", "bank", []string{"AU"}},
}

var (
	paymentCatalogOnce []paymentMethod
	paymentCatalogIdx  map[string]paymentMethod
)

// financePaymentMethods builds the full catalog: 3 generic rails per country
// (714) + the curated named methods (167) = 881 total.
func financePaymentMethods() []paymentMethod {
	if paymentCatalogOnce != nil {
		return paymentCatalogOnce
	}
	out := make([]paymentMethod, 0, 3*len(financeCountryCodes)+len(curatedPaymentMethods))
	for _, cc := range financeCountryCodes {
		lc := strings.ToLower(cc)
		out = append(out,
			paymentMethod{Code: "bank_transfer_" + lc, Name: "Bank Transfer (" + cc + ")", Kind: "bank", Countries: []string{cc}},
			paymentMethod{Code: "mobile_money_" + lc, Name: "Mobile Money (" + cc + ")", Kind: "mobile", Countries: []string{cc}},
			paymentMethod{Code: "instant_payment_" + lc, Name: "Instant Payment (" + cc + ")", Kind: "mobile", Countries: []string{cc}},
		)
	}
	out = append(out, curatedPaymentMethods...)
	paymentCatalogOnce = out
	paymentCatalogIdx = make(map[string]paymentMethod, len(out))
	for _, m := range out {
		paymentCatalogIdx[m.Code] = m
	}
	return out
}

func financeCountries() []string { return financeCountryCodes }

func paymentMethodByCode(code string) (paymentMethod, bool) {
	financePaymentMethods()
	m, ok := paymentCatalogIdx[code]
	return m, ok
}

// paymentMethodAvailableIn reports whether a method can be used in a country.
func paymentMethodAvailableIn(code, country string) bool {
	m, ok := paymentMethodByCode(code)
	if !ok {
		return false
	}
	if len(m.Countries) == 0 {
		return true // global rail
	}
	country = strings.ToUpper(country)
	for _, cc := range m.Countries {
		if cc == country {
			return true
		}
	}
	return false
}
