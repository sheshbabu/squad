import SettingsSection from '../../commons/components/SettingsSection.jsx';
import SettingsRow from '../../commons/components/SettingsRow.jsx';
import Toggle from '../../commons/components/Toggle.jsx';
import Button from '../../commons/components/Button.jsx';
import showToast from '../../commons/components/Toast.jsx';
import ToolGroupIcon from '../../commons/components/ToolGroupIcon.jsx';
import { openModal } from '../../commons/components/Modal.jsx';
import ApiClient from '../../commons/http/ApiClient.js';
import ConnectionModal from '../connections/ConnectionModal.jsx';
import './BotToolsPane.css';

const { useState, useEffect } = React;

export default function BotToolsPane({ bot, onSaved }) {
  const [toolGroups, setToolGroups] = useState([]);
  const [connections, setConnections] = useState([]);
  const [disabledToolGroups, setDisabledToolGroups] = useState(bot.disabledToolGroups);
  const [isSaving, setIsSaving] = useState(false);

  useEffect(() => {
    ApiClient.getToolGroups().then(setToolGroups);
  }, []);

  useEffect(() => {
    ApiClient.getConnections(bot.botId).then(setConnections);
  }, [bot.botId]);

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

    ApiClient.updateBotTools(bot.botId, { disabledToolGroups })
      .then(saved => {
        showToast("Saved");
        onSaved(saved);
      })
      .finally(() => setIsSaving(false));
  }

  function handleAddConnectionClick(group) {
    openModal(<ConnectionModal botId={bot.botId} group={group} onSaved={handleConnectionSaved} />);
  }

  function handleEditConnectionClick(group, connection) {
    openModal(<ConnectionModal botId={bot.botId} group={group} connection={connection} onSaved={handleConnectionSaved} onDeleted={handleConnectionDeleted} />);
  }

  function handleConnectionSaved(saved) {
    setConnections(previous => {
      const others = previous.filter(item => item.connectionId !== saved.connectionId);
      return [...others, saved].sort((a, b) => a.connectionId - b.connectionId);
    });
  }

  function handleConnectionDeleted(connectionId) {
    setConnections(previous => previous.filter(item => item.connectionId !== connectionId));
  }

  const groups = toolGroups.map(group => {
    const isEnabled = group.isAvailable && !disabledToolGroups.includes(group.groupId);
    const groupConnections = connections.filter(item => item.groupId === group.groupId);

    let description = group.description;
    if (!group.isAvailable) {
      description = group.setupHint;
    } else if (group.connection !== null && groupConnections.length === 0) {
      description = `Add a connection to use ${group.name}.`;
    }

    let connectionList = null;
    if (group.isAvailable && group.connection !== null) {
      const connectionItems = groupConnections.map(connection => (
        <div key={connection.connectionId} className="bot-tools-pane-connection" onClick={() => handleEditConnectionClick(group, connection)}>
          <span className="bot-tools-pane-connection-name">{connection.name}</span>
        </div>
      ));

      connectionList = (
        <div className="bot-tools-pane-connections">
          {connectionItems}
          <div className="bot-tools-pane-connection add" onClick={() => handleAddConnectionClick(group)}>
            <span className="bot-tools-pane-connection-name">Add {group.name} connection</span>
          </div>
        </div>
      );
    }

    return (
      <div key={group.groupId}>
        <SettingsRow icon={<ToolGroupIcon groupId={group.groupId} />} label={group.name} description={description}>
          <Toggle isEnabled={isEnabled} isDisabled={!group.isAvailable} onChange={value => handleToggleChange(group.groupId, value)} />
        </SettingsRow>
        {connectionList}
      </div>
    );
  });

  return (
    <div className="settings-pane bot-form bot-tools-pane">
      <form className="settings-form" onSubmit={handleSubmit}>
        <SettingsSection title="Tools" description="Choose which tools this bot can use. Tools that need keys stay off until they are set, and connections save as soon as you add them.">
          <div className="settings-card">{groups}</div>
        </SettingsSection>
        <Button type="submit" variant="primary" isDisabled={isSaving}>Save</Button>
      </form>
    </div>
  );
}
