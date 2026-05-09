const defaultTimeoutMs = 4500;

export async function fetchWithTimeout(input: Parameters<typeof fetch>[0], init: RequestInit = {}, timeoutMs = defaultTimeoutMs) {
  const controller = new AbortController();
  const timeout = globalThis.setTimeout(() => controller.abort(), timeoutMs);
  try {
    return await fetch(input, { ...init, signal: controller.signal });
  } catch (err) {
    if (controller.signal.aborted) throw new Error("Network timed out. Using saved cards if available.");
    throw err;
  } finally {
    globalThis.clearTimeout(timeout);
  }
}
