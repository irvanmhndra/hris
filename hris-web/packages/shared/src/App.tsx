import { QueryClientProvider } from "@tanstack/react-query";
import { BrowserRouter } from "react-router";
import { client } from "./components/common";
import { Shell } from "./layouts/Shell";
export function HRISApp({ mode }: { mode: "admin" | "employee" }) {
  return (
    <QueryClientProvider client={client}>
      <BrowserRouter>
        <Shell mode={mode} />
      </BrowserRouter>
    </QueryClientProvider>
  );
}
