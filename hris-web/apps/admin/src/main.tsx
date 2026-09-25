import React from "react";
import ReactDOM from "react-dom/client";
import { AdminApp } from "@hris/shared/admin";
import "@hris/shared/styles.css";
ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <AdminApp />
  </React.StrictMode>,
);
