import { QueryClientProvider } from "@tanstack/react-query";
import { lazy, type ComponentType, type ReactNode } from "react";
import { BrowserRouter } from "react-router";
import { client } from "./components/common";
import { Shell, type Mode, type NavItem } from "./layouts/Shell";
export function HRISApp({
  mode,
  links,
  children,
}: {
  mode: Mode;
  links: readonly NavItem[];
  children: ReactNode;
}) {
  return (
    <QueryClientProvider client={client}>
      <BrowserRouter>
        <Shell mode={mode} links={links}>
          {children}
        </Shell>
      </BrowserRouter>
    </QueryClientProvider>
  );
}
// lazyPage code-splits one named page export into its own chunk.
export function lazyPage<
  M extends Record<K, ComponentType<any>>,
  K extends keyof M,
>(load: () => Promise<M>, name: K) {
  return lazy(() => load().then((m) => ({ default: m[name] })));
}
