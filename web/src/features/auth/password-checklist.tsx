import { passwordRules } from "./registration-validation";

export function PasswordChecklist({
  password,
  confirmation,
}: {
  password: string;
  confirmation: string;
}) {
  return (
    <div id="password-requirements" className="password-checklist">
      <p className="mb-3 text-xs font-bold">شرایط رمز عبور</p>
      <ul className="space-y-2" aria-label="شرایط رمز عبور">
        {passwordRules(password, confirmation)
          .filter((rule) => rule.id !== "bytes")
          .map((rule) => (
            <li
              key={rule.id}
              data-rule={rule.id}
              data-passed={rule.passed}
              className={rule.passed ? "rule-passed" : "rule-pending"}
            >
              <span className="rule-icon" aria-hidden="true">
                {rule.passed ? "✓" : "×"}
              </span>
              <span>{rule.label}</span>
              <span className="sr-only">
                {rule.passed ? " — رعایت شده" : " — رعایت نشده"}
              </span>
            </li>
          ))}
      </ul>
    </div>
  );
}
