const strict = process.env.BAOTHEX_STRICT_PRODUCTION_CONFIG === "true";

if (strict) {
  const required = ["NEXT_PUBLIC_API_URL", "NEXT_PUBLIC_SITE_URL"];
  const missing = required.filter((name) => !process.env[name]);
  if (missing.length) {
    throw new Error(`Missing production configuration: ${missing.join(", ")}`);
  }
  for (const name of required) {
    const value = new URL(process.env[name]);
    const localHost = ["localhost", "127.0.0.1", "[::1]"].includes(value.hostname);
    if (value.protocol !== "https:" && !localHost) {
      throw new Error(`${name} must use https in production (received ${value.origin})`);
    }
  }
}
