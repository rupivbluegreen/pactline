export default function SignInSent() {
  return (
    <main className="flex min-h-screen flex-col items-center justify-center gap-4 p-8">
      <h1 className="text-2xl font-semibold">Check your email</h1>
      <p className="text-sm text-neutral-500">
        We sent a sign-in link. Open it on this device to continue.
      </p>
      <p className="text-xs text-neutral-400">
        (Dev: open Mailpit at{" "}
        <a className="underline" href="http://localhost:8025">
          localhost:8025
        </a>
        .)
      </p>
    </main>
  );
}
