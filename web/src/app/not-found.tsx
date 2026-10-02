import Link from "next/link";
export default function NotFound() {
  return (
    <section className="panel text-center">
      <p className="text-5xl font-bold text-blue-600">۴۰۴</p>
      <h1 className="mt-5 text-xl font-bold">این صفحه پیدا نشد</h1>
      <Link href="/home" className="primary mt-6">
        بازگشت به پیشخوان
      </Link>
    </section>
  );
}
