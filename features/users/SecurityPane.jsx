import Button from '../../commons/components/Button.jsx';
import Input from '../../commons/components/Input.jsx';
import SettingsRow from '../../commons/components/SettingsRow.jsx';
import SettingsSection from '../../commons/components/SettingsSection.jsx';
import { showToast } from '../../commons/components/Toast.jsx';
import { LogOutIcon } from '../../commons/components/Icon.jsx';
import ApiClient from '../../commons/http/ApiClient.js';

const { useState } = React;

export default function SecurityPane() {
  const [oldPassword, setOldPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [oldPasswordError, setOldPasswordError] = useState("");
  const [newPasswordError, setNewPasswordError] = useState("");
  const [confirmPasswordError, setConfirmPasswordError] = useState("");
  const [isPasswordLoading, setIsPasswordLoading] = useState(false);
  const [isLogoutLoading, setIsLogoutLoading] = useState(false);

  function handlePasswordSubmit(event) {
    event.preventDefault();

    setOldPasswordError("");
    setNewPasswordError("");
    setConfirmPasswordError("");

    if (oldPassword.trim() === "") {
      setOldPasswordError("Current password is required");
      return;
    }

    if (newPassword.trim() === "") {
      setNewPasswordError("New password is required");
      return;
    }

    if (newPassword !== confirmPassword) {
      setConfirmPasswordError("Passwords do not match");
      return;
    }

    if (oldPassword === newPassword) {
      setNewPasswordError("New password must be different");
      return;
    }

    setIsPasswordLoading(true);

    ApiClient.updatePassword({ oldPassword, newPassword })
      .then(() => {
        showToast("Password updated successfully");
        setOldPassword("");
        setNewPassword("");
        setConfirmPassword("");
      })
      .catch(error => {
        if (error.code === "INCORRECT_OLD_PASSWORD") {
          setOldPasswordError("Incorrect password");
        } else if (error.code === "INVALID_NEW_PASSWORD") {
          setNewPasswordError(error.message);
        }
      })
      .finally(() => setIsPasswordLoading(false));
  }

  // The service worker keeps cached chats for offline reading, so they are cleared on the way out.
  function handleLogoutClick() {
    setIsLogoutLoading(true);

    ApiClient.logout()
      .then(() => {
        navigator.serviceWorker?.controller?.postMessage({ type: "CLEAR_CACHE" });
        window.location.href = "/";
      })
      .finally(() => setIsLogoutLoading(false));
  }

  let passwordButtonText = "Update Password";
  if (isPasswordLoading) {
    passwordButtonText = "Updating...";
  }

  let logoutButtonText = "Log out";
  if (isLogoutLoading) {
    logoutButtonText = "Logging out...";
  }

  return (
    <div className="settings-pane">
      <SettingsSection title="Change Password" description="Update your account password. You'll need to provide your current password.">
        <form className="settings-form" onSubmit={handlePasswordSubmit}>
          <Input id="current-password" label="Current Password" type="password" placeholder="Enter current password" value={oldPassword} error={oldPasswordError} isDisabled={isPasswordLoading} onChange={event => setOldPassword(event.target.value)} />
          <Input id="new-password" label="New Password" type="password" placeholder="Enter new password" value={newPassword} error={newPasswordError} isDisabled={isPasswordLoading} onChange={event => setNewPassword(event.target.value)} />
          <Input id="confirm-password" label="Confirm New Password" type="password" placeholder="Confirm new password" value={confirmPassword} error={confirmPasswordError} isDisabled={isPasswordLoading} onChange={event => setConfirmPassword(event.target.value)} />
          <Button type="submit" variant="primary" isDisabled={isPasswordLoading}>{passwordButtonText}</Button>
        </form>
      </SettingsSection>

      <SettingsSection title="Session" description="Manage how you're signed in on this device.">
        <div className="settings-card">
          <SettingsRow icon={<LogOutIcon />} label="Current session" description="Logging out means you'll need to sign in again.">
            <Button onClick={handleLogoutClick} isDisabled={isLogoutLoading}>{logoutButtonText}</Button>
          </SettingsRow>
        </div>
      </SettingsSection>
    </div>
  );
}
