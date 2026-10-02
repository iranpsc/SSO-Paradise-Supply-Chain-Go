package application

import (
	"context"
	"github.com/iranpsc/SSO-Paradise-Supply-Chain-Go/internal/domain"
)

type Profiles interface {
	PersonalInfo(context.Context, int64) (domain.PersonalInfo, error)
	SavePersonalInfo(context.Context, domain.PersonalInfo, []domain.Media) error
	SaveAvatar(context.Context, domain.Media) error
	Media(context.Context, int64, string) (domain.Media, error)
	PublicProfile(context.Context, int64) (domain.PublicProfile, error)
}

type Profile struct{ Store Profiles }

func (p *Profile) Update(ctx context.Context, u domain.User, in domain.PersonalInfo, media []domain.Media) error {
	if u.EmailVerifiedAt == nil {
		return domain.ErrUnverified
	}
	if v := in.Validate(); len(v) > 0 {
		return v
	}
	in.UserID = u.ID
	required := []string{"melli_card_scan", "certificate_scan", "bank_card_scan"}
	uploaded := map[string]domain.Media{}
	for i := range media {
		kind := media[i].Kind
		known := false
		for _, k := range required {
			if k == kind {
				known = true
				break
			}
		}
		if !known {
			return domain.Validation{"files": "نوع مدرک معتبر نیست."}
		}
		ext := ".jpg"
		switch media[i].ContentType {
		case "image/png":
			ext = ".png"
		case "image/webp":
			ext = ".webp"
		case "image/jpeg":
			ext = ".jpg"
		}
		if msg := domain.ValidateImage(kind+ext, media[i].Data); msg != "" {
			return domain.Validation{kind: msg}
		}
		media[i].UserID = u.ID
		uploaded[kind] = media[i]
	}
	var toSave []domain.Media
	for _, k := range required {
		if m, ok := uploaded[k]; ok {
			toSave = append(toSave, m)
			continue
		}
		if _, err := p.Store.Media(ctx, u.ID, k); err != nil {
			return domain.Validation{k: "تصویر مدرک الزامی است."}
		}
	}
	return p.Store.SavePersonalInfo(ctx, in, toSave)
}

func (p *Profile) Get(ctx context.Context, u domain.User) (domain.PersonalInfo, map[string]bool, error) {
	var empty map[string]bool
	if u.EmailVerifiedAt == nil {
		return domain.PersonalInfo{}, empty, domain.ErrUnverified
	}
	info, err := p.Store.PersonalInfo(ctx, u.ID)
	if err != nil {
		return domain.PersonalInfo{}, empty, err
	}
	docs := map[string]bool{}
	for _, k := range []string{"melli_card_scan", "certificate_scan", "bank_card_scan"} {
		_, err := p.Store.Media(ctx, u.ID, k)
		docs[k] = err == nil
	}
	return info, docs, nil
}

func (p *Profile) UpdateAvatar(ctx context.Context, u domain.User, filename string, data []byte, contentType string) error {
	if u.EmailVerifiedAt == nil {
		return domain.ErrUnverified
	}
	if len(data) == 0 || len(data) > (1<<20) {
		return domain.Validation{"avatar": "حجم فایل نباید بیشتر از ۱ مگابایت باشد."}
	}
	if msg := domain.ValidateImage(filename, data); msg != "" {
		return domain.Validation{"avatar": msg}
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	return p.Store.SaveAvatar(ctx, domain.Media{UserID: u.ID, Kind: "avatars", ContentType: contentType, Data: data})
}
