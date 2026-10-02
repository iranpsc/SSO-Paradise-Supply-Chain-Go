const socialLinks = [
  {
    name: "اینستاگرام",
    label: "Instagram",
    href: "https://www.instagram.com/rgb.irpsc/",
  },
  { name: "یوتیوب", label: "YouTube", href: "https://www.youtube.com/@Irpsc" },
  {
    name: "آپارات",
    label: "Aparat",
    href: "https://www.aparat.com/Qzparadise.ir",
  },
  { name: "روبیکا", label: "Rubika", href: "https://rubika.ir/metaverse_iran" },
];
const licenseImages = [
  {
    src: "https://irpsc.com/img-icon/vezarat.png",
    label: "وزارت تعاون، کار و رفاه اجتماعی",
  },
  {
    src: "https://irpsc.com/img-icon/enamad.png",
    label: "نماد اعتماد الکترونیک",
  },
  {
    src: "https://irpsc.com/img-icon/qazaii.png",
    label: "ثبت اسناد و املاک کشور",
  },
];

export function FooterDetails() {
  return (
    <div className="holding-footer">
      <div className="footer-brand-row">
        <a
          href="https://irpsc.com"
          target="_blank"
          rel="noopener noreferrer"
          className="footer-brand"
        >
          <span className="footer-brand-icon" aria-hidden="true">
            <svg
              width="27"
              height="27"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="1.5"
            >
              <path d="m12 3 9 5-9 5-9-5 9-5Z" />
              <path d="m3 12 9 5 9-5M3 16l9 5 9-5" />
            </svg>
          </span>
          <span>
            <span className="footer-kicker">
              یکپارچگی در ارتباط، گستردگی در خدمات
            </span>
            <strong>زنجیره تامین بهشت</strong>
          </span>
        </a>
        <a
          href="https://irpsc.com"
          target="_blank"
          rel="noopener noreferrer"
          className="footer-site-link"
        >
          وب‌سایت هلدینگ <span aria-hidden="true">↗</span>
        </a>
      </div>
      <hr className="footer-rule" />
      <div className="footer-columns">
        <section aria-labelledby="holding-about">
          <span className="footer-section-number" aria-hidden="true">
            01 / ABOUT
          </span>
          <h2 id="holding-about">دربارهٔ هلدینگ</h2>
          <p className="footer-description">
            هلدینگ تعاونی‌های زنجیره تامین بهشت، مجموعه‌ای از سامانه‌ها و خدمات
            را در کنار هم قرار داده است. سامانهٔ احراز هویت مرکزی این مجموعه،
            دسترسی یکپارچه به خدمات زیرمجموعه را از طریق یک حساب کاربری فراهم
            می‌کند.
          </p>
          <span className="footer-signature">
            <span aria-hidden="true" /> یک حساب، دسترسی یکپارچه
          </span>
        </section>
        <section aria-labelledby="holding-social">
          <span className="footer-section-number" aria-hidden="true">
            02 / CONNECT
          </span>
          <h2 id="holding-social">همراه ما باشید</h2>
          <p className="footer-description">
            خبرها و محتوای مجموعه را در شبکه‌های اجتماعی دنبال کنید.
          </p>
          <ul className="footer-social-list">
            {socialLinks.map((link) => (
              <li key={link.href}>
                <a href={link.href} target="_blank" rel="noopener noreferrer">
                  <span>{link.name}</span>
                  <span className="footer-social-label" lang="en">
                    {link.label}
                  </span>
                  <span aria-hidden="true">↗</span>
                </a>
              </li>
            ))}
          </ul>
        </section>
        <section aria-labelledby="holding-licenses" className="footer-licenses">
          <span className="footer-section-number" aria-hidden="true">
            03 / LICENSES
          </span>
          <h2 id="holding-licenses">مجوزها و اعتبارنامه‌ها</h2>
          <ul className="footer-license-gallery" aria-label="تصاویر مجوزها">
            {licenseImages.map((item) => (
              <li key={item.src}>
                <figure>
                  <div className="footer-license-image">
                    <img
                      src={item.src}
                      alt={item.label}
                      width={64}
                      height={64}
                      loading="lazy"
                    />
                  </div>
                  <figcaption>{item.label}</figcaption>
                </figure>
              </li>
            ))}
          </ul>
        </section>
      </div>
      <hr className="footer-rule" />
      <div className="footer-bottom">
        <p>تمامی حقوق این سامانه متعلق به هلدینگ زنجیره تامین بهشت است.</p>
        <span>
          تونل زمان <span aria-hidden="true">/</span> سامانه احراز هویت مرکزی
        </span>
      </div>
    </div>
  );
}
