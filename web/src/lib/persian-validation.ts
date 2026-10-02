type Control = HTMLInputElement | HTMLSelectElement | HTMLTextAreaElement;
function control(target: EventTarget): Control | null {
  return target instanceof HTMLInputElement || target instanceof HTMLSelectElement || target instanceof HTMLTextAreaElement ? target : null;
}
export function localizeValidation(target: EventTarget) {
  const input = control(target);
  if (!input) return;
  const validity = input.validity;
  // Clear an earlier custom error before reading current native constraints.
  input.setCustomValidity("");
  if (input.validity.valid) return;
  let text = "مقدار واردشده معتبر نیست.";
  if (validity.valueMissing) text = "تکمیل این فیلد الزامی است.";
  else if (validity.typeMismatch) text = "قالب مقدار واردشده معتبر نیست.";
  else if (validity.patternMismatch) text = "مقدار واردشده با قالب موردنیاز مطابقت ندارد.";
  else if (validity.tooShort) text = "تعداد نویسه‌های واردشده کمتر از حد مجاز است.";
  else if (validity.tooLong) text = "تعداد نویسه‌های واردشده بیشتر از حد مجاز است.";
  else if (validity.rangeUnderflow || validity.rangeOverflow) text = "عدد واردشده خارج از محدودهٔ مجاز است.";
  else if (validity.stepMismatch || validity.badInput) text = "عدد واردشده معتبر نیست.";
  input.setCustomValidity(text);
}
export function clearValidation(target: EventTarget) { control(target)?.setCustomValidity(""); }
