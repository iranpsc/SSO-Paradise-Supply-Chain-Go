package httpapi

import (
	"encoding/json"
	"io"
	"net/http"
	"net/textproto"
	"strconv"
	"strings"

	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/domain"
)

var documentKinds = map[string]bool{
	"melli_card_scan":  true,
	"certificate_scan": true,
	"bank_card_scan":   true,
}

func parseCompanyBool(raw string) (*bool, bool) {
	v := strings.ToLower(strings.TrimSpace(raw))
	switch v {
	case "1", "true", "on", "yes":
		b := true
		return &b, true
	case "0", "false", "off", "no":
		b := false
		return &b, true
	default:
		return nil, false
	}
}

func sniff(data []byte) string {
	if len(data) == 0 {
		return "application/octet-stream"
	}
	// net/http.DetectContentType is enough; keep stdlib only.
	h := http.DetectContentType(data)
	if strings.HasPrefix(h, "image/") {
		// DetectContentType returns "image/jpeg", "image/png", "image/webp" etc.
		if i := strings.Index(h, ";"); i >= 0 {
			h = h[:i]
		}
		return h
	}
	return h
}

func (s *Server) requireProfiles() bool { return s.profiles != nil }

func (s *Server) personalInfo(w http.ResponseWriter, r *http.Request, u domain.User) {
	if !s.requireProfiles() {
		s.fail(w, domain.ErrNotFound)
		return
	}
	info, docs, err := s.profiles.Get(r.Context(), u)
	if err != nil {
		s.fail(w, err)
		return
	}
	respond(w, 200, map[string]any{"data": info, "documents": docs})
}

func personalInfoFromJSON(dst *domain.PersonalInfo, raw []byte) error {
	return json.Unmarshal(raw, dst)
}

func (s *Server) updatePersonalInfo(w http.ResponseWriter, r *http.Request, u domain.User) {
	if !s.requireProfiles() {
		s.fail(w, domain.ErrNotFound)
		return
	}
	ct := r.Header.Get("Content-Type")
	var info domain.PersonalInfo
	var media []domain.Media
	if strings.HasPrefix(ct, "multipart/form-data") {
		r.Body = http.MaxBytesReader(w, r.Body, 7<<20)
		if err := r.ParseMultipartForm(6 << 20); err != nil {
			respond(w, 400, map[string]string{"message": "فرم معتبر نیست."})
			return
		}
		defer r.MultipartForm.RemoveAll()
		isCompany, ok := parseCompanyBool(r.FormValue("is_company"))
		if !ok {
			s.fail(w, domain.Validation{"is_company": "نوع حساب الزامی است."})
			return
		}
		info = domain.PersonalInfo{
			IsCompany:                 isCompany,
			FirstName:                 strings.TrimSpace(r.FormValue("first_name")),
			LastName:                  strings.TrimSpace(r.FormValue("last_name")),
			Mobile:                    strings.TrimSpace(r.FormValue("mobile")),
			Telephone:                 strings.TrimSpace(r.FormValue("telephone")),
			NationalCode:              strings.TrimSpace(r.FormValue("national_code")),
			Address:                   strings.TrimSpace(r.FormValue("address")),
			CompanyName:               strings.TrimSpace(r.FormValue("company_name")),
			CompanyAddress:            strings.TrimSpace(r.FormValue("company_address")),
			CompanyRegistrationNumber: strings.TrimSpace(r.FormValue("company_registration_number")),
			CompanyNationalNumber:     strings.TrimSpace(r.FormValue("company_national_number")),
			CompanyTaxNumber:          strings.TrimSpace(r.FormValue("company_tax_number")),
			CompanyExecutiveName:      strings.TrimSpace(r.FormValue("company_executive_name")),
		}
		for kind := range documentKinds {
			f, hdr, err := r.FormFile(kind)
			if err != nil {
				continue
			}
			data, err := io.ReadAll(io.LimitReader(f, domain.MaxDocumentBytes+1))
			_ = f.Close()
			if err != nil {
				s.fail(w, domain.Validation{kind: "خواندن فایل ممکن نشد."})
				return
			}
			if len(data) == 0 {
				s.fail(w, domain.Validation{kind: "تصویر مدرک الزامی است."})
				return
			}
			if msg := domain.ValidateDocument(hdr.Filename, data); msg != "" {
				s.fail(w, domain.Validation{kind: msg})
				return
			}
			contentType := sniff(data)
			media = append(media, domain.Media{Kind: kind, ContentType: contentType, Data: data})
		}
	} else {
		var in domain.PersonalInfo
		if !decode(w, r, &in) {
			return
		}
		info = in
	}
	if err := s.profiles.Update(r.Context(), u, info, media); err != nil {
		s.fail(w, err)
		return
	}
	updated, docs, err := s.profiles.Get(r.Context(), u)
	if err != nil {
		s.fail(w, err)
		return
	}
	respond(w, 200, map[string]any{"data": updated, "documents": docs, "message": "اطلاعات با موفقیت ثبت شد."})
}

func (s *Server) document(w http.ResponseWriter, r *http.Request, u domain.User) {
	if !s.requireProfiles() {
		s.fail(w, domain.ErrNotFound)
		return
	}
	kind := r.PathValue("kind")
	if !documentKinds[kind] {
		s.fail(w, domain.ErrNotFound)
		return
	}
	m, err := s.profiles.Store.Media(r.Context(), u.ID, kind)
	if err != nil {
		s.fail(w, err)
		return
	}
	w.Header().Set("Content-Type", m.ContentType)
	w.Header().Set("Content-Length", strconv.Itoa(len(m.Data)))
	w.Header().Set("Cache-Control", "private, no-store")
	_, _ = w.Write(m.Data)
}

func (s *Server) ownAvatar(w http.ResponseWriter, r *http.Request, u domain.User) {
	if !s.requireProfiles() {
		s.fail(w, domain.ErrNotFound)
		return
	}
	m, err := s.profiles.Store.Media(r.Context(), u.ID, "avatars")
	if err != nil {
		s.fail(w, err)
		return
	}
	w.Header().Set("Content-Type", m.ContentType)
	w.Header().Set("Content-Length", strconv.Itoa(len(m.Data)))
	w.Header().Set("Cache-Control", "private, no-store")
	_, _ = w.Write(m.Data)
}

func (s *Server) updateAvatar(w http.ResponseWriter, r *http.Request, u domain.User) {
	if !s.requireProfiles() {
		s.fail(w, domain.ErrNotFound)
		return
	}
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
		respond(w, 415, map[string]string{"message": "برای ارسال تصویر، از فرم بارگذاری فایل استفاده کنید."})
		return
	}
	if err := r.ParseMultipartForm(2 << 20); err != nil {
		respond(w, 400, map[string]string{"message": "فرم معتبر نیست."})
		return
	}
	f, hdr, err := r.FormFile("avatar")
	if err != nil {
		s.fail(w, domain.Validation{"avatar": "فایل آواتار الزامی است."})
		return
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, (1<<20)+1))
	if err != nil {
		s.fail(w, domain.Validation{"avatar": "خواندن فایل ممکن نشد."})
		return
	}
	filename := hdr.Filename
	if filename == "" {
		filename = "avatar.jpg"
	}
	// Validate content-type header spoofing via sniffed type for storage.
	stored := sniff(data)
	if err := s.profiles.UpdateAvatar(r.Context(), u, filename, data, stored); err != nil {
		s.fail(w, err)
		return
	}
	_ = textproto.MIMEHeader{}
	respond(w, 200, map[string]string{"message": "آواتار به‌روزرسانی شد."})
}

func (s *Server) publicAvatar(w http.ResponseWriter, r *http.Request) {
	if !s.requireProfiles() {
		s.fail(w, domain.ErrNotFound)
		return
	}
	id, err := strconv.ParseInt(r.PathValue("user"), 10, 64)
	if err != nil || id < 1 {
		s.fail(w, domain.ErrNotFound)
		return
	}
	m, err := s.profiles.Store.Media(r.Context(), id, "avatars")
	if err != nil {
		s.fail(w, err)
		return
	}
	w.Header().Set("Content-Type", m.ContentType)
	w.Header().Set("Content-Length", strconv.Itoa(len(m.Data)))
	w.Header().Set("Cache-Control", "public, max-age=3600")
	_, _ = w.Write(m.Data)
}
