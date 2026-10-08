import { BrowserRouter, Route, Routes } from "react-router-dom";
import { Layout } from "./components/Layout";
import { BatchesPage } from "./pages/BatchesPage";
import { HomePage } from "./pages/HomePage";
import { InspectorPage } from "./pages/InspectorPage";
import { LandingPage } from "./pages/LandingPage";
import { NotFoundPage } from "./pages/NotFoundPage";
import { OtpsPage } from "./pages/OtpsPage";
import { SettingsPage } from "./pages/SettingsPage";

export function AppRoutes() {
  return (
    <BrowserRouter>
      <Routes>
        {/* Full-width standalone marketing landing page */}
        <Route path="/" element={<LandingPage />} />

        {/* Local sandbox web app shell */}
        <Route element={<Layout />}>
          <Route path="/inbox" element={<HomePage />} />
          <Route path="/otps" element={<OtpsPage />} />
          <Route path="/batches" element={<BatchesPage />} />
          <Route path="/inspector" element={<InspectorPage />} />
          <Route path="/settings" element={<SettingsPage />} />
          <Route path="*" element={<NotFoundPage />} />
        </Route>
      </Routes>
    </BrowserRouter>
  );
}
