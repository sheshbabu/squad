import SettingsSection from '../../commons/components/SettingsSection.jsx';
import SettingsRow from '../../commons/components/SettingsRow.jsx';
import Toggle from '../../commons/components/Toggle.jsx';
import Input from '../../commons/components/Input.jsx';
import Button from '../../commons/components/Button.jsx';
import showToast from '../../commons/components/Toast.jsx';
import ToolGroupIcon from '../../commons/components/ToolGroupIcon.jsx';
import ApiClient from '../../commons/http/ApiClient.js';

const { useState, useEffect } = React;

export default function BotToolsPane({ bot, onSaved }) {
  const [toolGroups, setToolGroups] = useState([]);
  const [disabledToolGroups, setDisabledToolGroups] = useState(bot.disabledToolGroups);
  const [zenApiToken, setZenApiToken] = useState(bot.zenApiToken);
  const [isSaving, setIsSaving] = useState(false);

  useEffect(() => {
    ApiClient.getToolGroups().then(setToolGroups);
  }, []);

  function handleToggleChange(groupId, isEnabled) {
    if (isEnabled) {
      setDisabledToolGroups(previous => previous.filter(item => item !== groupId));
    } else {
      setDisabledToolGroups(previous => [...previous, groupId]);
    }
  }

  function handleSubmit(event) {
    event.preventDefault();
    setIsSaving(true);

    ApiClient.updateBotTools(bot.botId, { disabledToolGroups, zenApiToken })
      .then(saved => {
        showToast("Saved");
        onSaved(saved);
      })
      .finally(() => setIsSaving(false));
  }

  const rows = toolGroups.map(group => {
    const isEnabled = group.isAvailable && !disabledToolGroups.includes(group.groupId);

    let description = group.description;
    if (!group.isAvailable) {
      description = group.setupHint;
    } else if (group.groupId === "zen" && zenApiToken.trim() === "") {
      description = "Add an API token below to use Zen.";
    }

    return (
      <SettingsRow key={group.groupId} icon={<ToolGroupIcon groupId={group.groupId} />} label={group.name} description={description}>
        <Toggle isEnabled={isEnabled} isDisabled={!group.isAvailable} onChange={value => handleToggleChange(group.groupId, value)} />
      </SettingsRow>
    );
  });

  let zenSection = null;
  const zenGroup = toolGroups.find(group => group.groupId === "zen");
  if (zenGroup !== undefined && zenGroup.isAvailable) {
    zenSection = (
      <SettingsSection title="Zen" description="Create a token in Zen under Settings, API Tokens. Its tag scopes decide which notes this bot can read and write.">
        <Input id="bot-zen-api-token" label="API token" type="password" placeholder="Paste a Zen API token" value={zenApiToken} onChange={event => setZenApiToken(event.target.value)} />
      </SettingsSection>
    );
  }

  return (
    <div className="settings-pane bot-form">
      <form className="settings-form" onSubmit={handleSubmit}>
        <SettingsSection title="Tools" description="Choose which tools this bot can use. Tools that need keys stay off until they are set.">
          <div className="settings-card">{rows}</div>
        </SettingsSection>
        {zenSection}
        <Button type="submit" variant="primary" isDisabled={isSaving}>Save</Button>
      </form>
    </div>
  );
}
