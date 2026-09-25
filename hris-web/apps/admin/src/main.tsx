import React from "react";
import ReactDOM from "react-dom/client";
import { HRISApp } from "@hris/shared";
import "@hris/shared/styles.css";
ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <HRISApp mode="admin" />
  </React.StrictMode>,
);
