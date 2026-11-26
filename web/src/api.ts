// The JSON API client. Errors arrive as {error: {code, message}}.

export class ApiError extends Error {
  constructor(
    readonly status: number,
    readonly code: string,
    message: string,
  ) {
    super(message);
  }
}

async function request<T>(
  method: string,
  path: string,
  body?: unknown,
): Promise<T> {
  const res = await fetch(`/api/v1${path}`, {
    method,
    credentials: "same-origin",
    headers:
      body === undefined ? undefined : { "Content-Type": "application/json" },
    body: body === undefined ? undefined : JSON.stringify(body),
  });
  if (!res.ok) {
    const fallback = {
      code: "internal",
      message: "Something went wrong. Please try again.",
    };
    const parsed = (await res.json().catch(() => null)) as {
      error?: typeof fallback;
    } | null;
    const err = parsed?.error ?? fallback;
    throw new ApiError(res.status, err.code, err.message);
  }
  const text = await res.text();
  return (text ? JSON.parse(text) : undefined) as T;
}

export type Me = {
  email: string;
  emailVerified: boolean;
  plan: "free" | "premium";
  timezone: string;
  pendingTimezone: string | null;
  firstDay: string;
  today: string;
  graceUntil: string | null;
  signInMethods: ("password" | "google")[];
};

export const getMe = () => request<Me>("GET", "/me");

export const signup = (email: string, password: string) =>
  request<void>("POST", "/auth/signup", {
    email,
    password,
    timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
  });

export const login = (email: string, password: string) =>
  request<void>("POST", "/auth/login", { email, password });

export const logout = () => request<void>("POST", "/auth/logout");

export const isUnauthenticated = (err: unknown) =>
  err instanceof ApiError && err.code === "unauthenticated";
