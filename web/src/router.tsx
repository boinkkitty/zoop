import { QueryClient, queryOptions } from "@tanstack/react-query";
import {
  createRootRouteWithContext,
  createRoute,
  createRouter,
  Outlet,
  redirect,
} from "@tanstack/react-router";
import { getMe, isUnauthenticated } from "./api";
import { AuthScreen } from "./auth/AuthScreen";
import {
  ForgotScreen,
  ResetScreen,
  VerifyScreen,
} from "./auth/EmailLinkScreens";
import { pageQuery } from "./page/queries";

export const queryClient = new QueryClient({
  defaultOptions: { queries: { retry: false, staleTime: 60_000 } },
});

export const meQuery = queryOptions({ queryKey: ["me"], queryFn: getMe });

// The signed-in user, or a redirect to sign in. The server decides "today", never the client.
async function requireMe(qc: QueryClient) {
  try {
    return await qc.ensureQueryData(meQuery);
  } catch (err) {
    if (isUnauthenticated(err)) throw redirect({ to: "/signin" });
    throw err;
  }
}

const rootRoute = createRootRouteWithContext<{ queryClient: QueryClient }>()({
  component: Outlet,
});

const indexRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/",
  beforeLoad: async ({ context }) => {
    const me = await requireMe(context.queryClient);
    throw redirect({
      to: "/day/$day/$notebook",
      params: { day: me.today, notebook: "my-journal" },
    });
  },
});

const signinRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/signin",
  validateSearch: (search: Record<string, unknown>): { error?: string } =>
    typeof search.error === "string" ? { error: search.error } : {},
  component: function SignIn() {
    const { error } = signinRoute.useSearch();
    return <AuthScreen mode="signin" googleError={error} />;
  },
});

const signupRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/signup",
  component: () => <AuthScreen mode="signup" />,
});

const tokenSearch = (search: Record<string, unknown>): { token: string } => ({
  token: typeof search.token === "string" ? search.token : "",
});

// The emailed links land here. Each POSTs its token, so a scanner's GET of the link can't use it up.
const verifyRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/verify",
  validateSearch: tokenSearch,
  component: function Verify() {
    return <VerifyScreen token={verifyRoute.useSearch().token} />;
  },
});

const resetRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/reset",
  validateSearch: tokenSearch,
  component: function Reset() {
    return <ResetScreen token={resetRoute.useSearch().token} />;
  },
});

const forgotRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/forgot",
  component: ForgotScreen,
});

// Lazy, so the auth screens never download the signed-in app (or, later, the editor).
const dayRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/day/$day/$notebook",
  beforeLoad: ({ context }) => requireMe(context.queryClient),
  loader: ({ context, params }) =>
    context.queryClient.ensureQueryData(pageQuery(params.day, params.notebook)),
}).lazy(() => import("./journal/JournalRoute").then((m) => m.Route));

// Unknown paths go home, which sends the visitor to sign in or to today.
const notFoundRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "$",
  beforeLoad: () => {
    throw redirect({ to: "/" });
  },
});

export const router = createRouter({
  routeTree: rootRoute.addChildren([
    indexRoute,
    signinRoute,
    signupRoute,
    verifyRoute,
    resetRoute,
    forgotRoute,
    dayRoute,
    notFoundRoute,
  ]),
  context: { queryClient },
});

declare module "@tanstack/react-router" {
  interface Register {
    router: typeof router;
  }
}
