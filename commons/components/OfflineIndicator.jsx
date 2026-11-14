import useIsOnline from "../utils/useIsOnline.js";
import "./OfflineIndicator.css";

export default function OfflineIndicator() {
  const isOnline = useIsOnline();

  if (isOnline) {
    return null;
  }

  return (
    <div className="offline-indicator">offline read-only view</div>
  );
}
