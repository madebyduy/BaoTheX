// Optional Sentry wiring for the Next.js server. It is a no-op unless
// SENTRY_DSN is set AND @sentry/nextjs is installed (`npm i @sentry/nextjs`),
// so the build never breaks when monitoring is not configured. The specifier is
// kept in a variable and marked webpackIgnore so the bundler does not try to
// resolve the (possibly absent) package at build time.
// Typed as string (not the string literal) so tsc does not try to resolve the
// optional module when it is not installed.
const SENTRY: string = "@sentry/nextjs";

export async function register() {
  if (!process.env.SENTRY_DSN) return;
  try {
    const Sentry = await import(/* webpackIgnore: true */ SENTRY);
    Sentry.init({
      dsn: process.env.SENTRY_DSN,
      tracesSampleRate: Number(process.env.SENTRY_TRACES_SAMPLE_RATE || "0.1"),
      environment: process.env.NODE_ENV,
    });
  } catch {
    // @sentry/nextjs not installed — monitoring stays off.
  }
}

export async function onRequestError(error: unknown, request: unknown, context: unknown) {
  if (!process.env.SENTRY_DSN) return;
  try {
    const Sentry = await import(/* webpackIgnore: true */ SENTRY);
    Sentry.captureRequestError?.(error, request, context);
  } catch {
    // no-op
  }
}
