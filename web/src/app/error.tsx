"use client";
export default function ErrorPage({ reset }: { reset: () => void }) {
  return (
    <section role="alert" className="panel text-center">
      <h1 className="text-xl font-bold">بارگذاری صفحه انجام نشد</h1>
      <button className="primary mt-6" onClick={reset}>
        تلاش دوباره
      </button>
    </section>
  );
}
