"use client";
import { useEffect, useRef, useState, type ReactNode } from "react";
import { usePathname } from "next/navigation";
import { AmbientParticles } from "./ambient-particles";
import { LegacyHeader } from "./legacy-header";
import { LegacyFooter } from "./legacy-footer";
import { clearValidation, localizeValidation } from "@/lib/persian-validation";
export function AppShell({ children }: { children: ReactNode }) {
  const [open, setOpen] = useState(false);
  const path = usePathname();
  const authPage = [
    "/login",
    "/register",
    "/password/email",
    "/password/reset",
    "/email/verify",
  ].includes(path);
  const container = useRef<HTMLDivElement>(null);
  function setTheme(dark: boolean) {
    document.documentElement.classList.toggle("dark", dark);
    try {
      localStorage.setItem("dark-mode", String(dark));
    } catch {}
  }
  useEffect(() => {
    try {
      document.documentElement.classList.toggle(
        "dark",
        localStorage.getItem("dark-mode") === "true",
      );
    } catch {}
  }, []);
  useEffect(() => {
    setOpen(false);
  }, [path]);
  useEffect(() => {
    if (!open) return;
    const previous = document.activeElement as HTMLElement | null;
    const nav = container.current?.querySelector<HTMLElement>("#open00");
    nav?.querySelector<HTMLElement>("a, button, [tabindex='0']")?.focus();
    function keyboard(event: KeyboardEvent) {
      if (event.key === "Escape") setOpen(false);
      if (event.key === "Tab" && window.innerWidth < 1024 && nav) {
        const items = Array.from(
          nav.querySelectorAll<HTMLElement>(
            "a[href],button,summary,input,[tabindex='0']",
          ),
        ).filter((el) => el.getClientRects().length > 0);
        const first = items[0],
          last = items.at(-1);
        if (event.shiftKey && document.activeElement === first) {
          event.preventDefault();
          last?.focus();
        } else if (!event.shiftKey && document.activeElement === last) {
          event.preventDefault();
          first?.focus();
        }
      }
    }
    document.addEventListener("keydown", keyboard);
    return () => {
      document.removeEventListener("keydown", keyboard);
      previous?.focus();
    };
  }, [open]);
  return (
    <div ref={container} className="legacy-shell" data-expanded={open}
      onInvalidCapture={event => localizeValidation(event.target)}
      onInputCapture={event => clearValidation(event.target)}
      onChangeCapture={event => clearValidation(event.target)}>
      <a href="#main-content" className="skip-link">
        رفتن به محتوای اصلی
      </a>
      <LegacyHeader
        openNav={() => setOpen(true)}
        closeNav={() => setOpen(false)}
        setTheme={setTheme}
      />
      {open && (
        <button
          aria-label="بستن فهرست کناری"
          className="fixed inset-0 z-[4000] bg-black/40 lg:hidden"
          onClick={() => setOpen(false)}
        />
      )}
      <main id="main-content" className="page-frame" tabIndex={-1}>
        <div
          className={
            authPage
              ? "content-surface auth-surface px-4 pt-24 pb-10 sm:px-6 lg:pt-10"
              : "content-surface px-4 pt-24 pb-10 sm:px-6 lg:pt-10"
          }
        >
          <AmbientParticles />
          <div
            className={
              authPage
                ? "content-foreground"
                : "content-foreground mx-auto max-w-6xl"
            }
          >
            {children}
          </div>
        </div>
        <LegacyFooter />
      </main>
    </div>
  );
}
