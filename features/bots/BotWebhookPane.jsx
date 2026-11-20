import SettingsSection from '../../commons/components/SettingsSection.jsx';
import Button from '../../commons/components/Button.jsx';
import ConfirmModal from '../../commons/components/ConfirmModal.jsx';
import { openModal } from '../../commons/components/Modal.jsx';
import showToast from '../../commons/components/Toast.jsx';
import ApiClient from '../../commons/http/ApiClient.js';
import { copyText, isClipboardAvailable } from '../../commons/utils/clipboard.js';
import './BotWebhookPane.css';

const { useState } = React;

export default function BotWebhookPane({ bot, onSaved }) {
  const [isSaving, setIsSaving] = useState(false);

  function save(promise) {
    setIsSaving(true);
    promise
      .then(onSaved)
      .finally(() => setIsSaving(false));
  }

  function handleCreateClick() {
    save(ApiClient.regenerateBotWebhook(bot.botId));
  }

  function handleRegenerateClick() {
    openModal(<ConfirmModal title="Regenerate URL" description="The current URL will stop working." confirmLabel="Regenerate" onConfirm={() => save(ApiClient.regenerateBotWebhook(bot.botId))} />);
  }

  function handleTurnOffClick() {
    openModal(<ConfirmModal title="Turn off webhook" description="The current URL will stop working." confirmLabel="Turn off" onConfirm={() => save(ApiClient.deleteBotWebhook(bot.botId))} />);
  }

  const url = `${window.location.origin}/api/hooks/${bot.webhookToken}/`;

  function handleCopyClick() {
    copyText(url, "Couldn't copy URL.", () => showToast("Copied"));
  }

  const description = `Anything sent to this URL starts a new chat with ${bot.name}, for example from an iOS Shortcut or a script. Anyone with the URL can message the bot, so keep it private.`;

  if (bot.webhookToken === "") {
    return (
      <div className="settings-pane bot-webhook-pane">
        <SettingsSection title="Webhook" description={description}>
          <Button variant="primary" onClick={handleCreateClick} isDisabled={isSaving}>Create URL</Button>
        </SettingsSection>
      </div>
    );
  }

  let copyButton = null;
  if (isClipboardAvailable()) {
    copyButton = <Button onClick={handleCopyClick}>Copy</Button>;
  }

  return (
    <div className="settings-pane bot-webhook-pane">
      <SettingsSection title="Webhook" description={description}>
        <div className="bot-webhook-pane-details">
          <code className="bot-webhook-pane-url">{url}</code>
          <pre className="bot-webhook-pane-example">{`curl -X POST ${url} -d "Your message"`}</pre>
          <div className="bot-webhook-pane-actions">
            {copyButton}
            <Button onClick={handleRegenerateClick} isDisabled={isSaving}>Regenerate</Button>
            <Button onClick={handleTurnOffClick} isDisabled={isSaving}>Turn off</Button>
          </div>
        </div>
      </SettingsSection>
    </div>
  );
}
