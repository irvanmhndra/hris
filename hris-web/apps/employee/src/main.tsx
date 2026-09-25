import React from "react";
import ReactDOM from "react-dom/client";
import { EmployeeApp } from "@hris/shared/employee";
import "@hris/shared/styles.css";
ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <EmployeeApp />
  </React.StrictMode>,
);
