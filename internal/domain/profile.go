package domain

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

type PersonalInfo struct {
	UserID                    int64  `json:"user_id"`
	IsCompany                 *bool  `json:"is_company"`
	FirstName                 string `json:"first_name"`
	LastName                  string `json:"last_name"`
	Mobile                    string `json:"mobile"`
	Telephone                 string `json:"telephone"`
	NationalCode              string `json:"national_code"`
	Address                   string `json:"address"`
	CompanyName               string `json:"company_name"`
	CompanyAddress            string `json:"company_address"`
	CompanyRegistrationNumber string `json:"company_registration_number"`
	CompanyNationalNumber     string `json:"company_national_number"`
	CompanyTaxNumber          string `json:"company_tax_number"`
	CompanyExecutiveName      string `json:"company_executive_name"`
	IsVerified                bool   `json:"is_verified"`
	VerificationMessages      string `json:"verification_messages"`
}

var mobilePattern = regexp.MustCompile(`^09[0-9]{9}$`)
var phonePattern = regexp.MustCompile(`^0[1-8]{2}[0-9]{8}$`)
var nationalPattern = regexp.MustCompile(`^[0-9]{8,10}$`)

func ValidNationalCode(value string) bool {
	if !nationalPattern.MatchString(value) {
		return false
	}
	value = strings.Repeat("0", 10-len(value)) + value
	sum := 0
	for i := 0; i < 9; i++ {
		sum += int(value[i]-'0') * (10 - i)
	}
	check := sum % 11
	if check >= 2 {
		check = 11 - check
	}
	return check == int(value[9]-'0')
}

func (p PersonalInfo) Validate() Validation {
	v := Validation{}
	if p.IsCompany == nil {
		v["is_company"] = "نوع حساب الزامی است."
	}
	for key, value := range map[string]string{"first_name": p.FirstName, "last_name": p.LastName, "address": p.Address} {
		if strings.TrimSpace(value) == "" || utf8.RuneCountInString(value) > 255 {
			v[key] = "این فیلد الزامی است و حداکثر ۲۵۵ نویسه دارد."
		}
	}
	if !mobilePattern.MatchString(p.Mobile) {
		v["mobile"] = "شماره موبایل معتبر وارد کنید."
	}
	if !phonePattern.MatchString(p.Telephone) {
		v["telephone"] = "شماره تلفن همراه با کد شهر وارد کنید."
	}
	if !ValidNationalCode(p.NationalCode) {
		v["national_code"] = "کد ملی معتبر وارد کنید."
	}
	for key, value := range map[string]string{"company_name": p.CompanyName, "company_address": p.CompanyAddress, "company_registration_number": p.CompanyRegistrationNumber, "company_national_number": p.CompanyNationalNumber, "company_tax_number": p.CompanyTaxNumber, "company_executive_name": p.CompanyExecutiveName} {
		if (p.IsCompany != nil && *p.IsCompany && strings.TrimSpace(value) == "") || utf8.RuneCountInString(value) > 255 {
			v[key] = "اطلاعات شرکت معتبر نیست."
		}
	}
	for key, value := range map[string]string{"company_registration_number": p.CompanyRegistrationNumber, "company_national_number": p.CompanyNationalNumber, "company_tax_number": p.CompanyTaxNumber} {
		if utf8.RuneCountInString(value) > 32 {
			v[key] = "حداکثر طول شناسه شرکت ۳۲ نویسه است."
		}
	}
	return v
}

type Media struct {
	LegacyPath     string `json:"-"`
	LegacyAbsolute bool   `json:"-"`
	UserID         int64  `json:"-"`
	Kind           string `json:"kind"`
	ContentType    string `json:"content_type"`
	Data           []byte `json:"-"`
}

type PublicProfile struct {
	AvatarAbsolute bool    `json:"-"`
	ID             int64   `json:"id"`
	Name           string  `json:"name"`
	Code           *string `json:"code"`
	Avatar         string  `json:"avatar"`
}
