import SidePaneModal from '../../commons/components/SidePaneModal.jsx';
import SettingsSection from '../../commons/components/SettingsSection.jsx';
import SettingsRow from '../../commons/components/SettingsRow.jsx';
import BotAvatar from '../../commons/components/BotAvatar.jsx';
import Button from '../../commons/components/Button.jsx';
import { openModal } from '../../commons/components/Modal.jsx';
import { BotIcon, NewIcon, ThemeIcon, SecurityIcon } from '../../commons/components/Icon.jsx';
import BotSettingsModal from '../bots/BotSettingsModal.jsx';
import AppearancePane from './AppearancePane.jsx';
import SecurityPane from './SecurityPane.jsx';
import './SettingsModal.css';

// Modals render in their own React root, outside the app context, so the bots and refresh callbacks come in as props.
export default function SettingsModal({ bots, onBotCreated, onBotSaved, onBotDeleted }) {
  const tabGroups = [
    [
      { id: "bots", label: "Bots", icon: <BotIcon />, content: <BotsPane bots={bots} onBotCreated={onBotCreated} onBotSaved={onBotSaved} onBotDeleted={onBotDeleted} /> },
      { id: "appearance", label: "Appearance", icon: <ThemeIcon />, content: <AppearancePane /> },
    ],
    [
      { id: "security", label: "Security", icon: <SecurityIcon />, content: <SecurityPane /> },
    ],
  ];

  return <SidePaneModal title="Settings" tabGroups={tabGroups} />;
}

function BotsPane({ bots, onBotCreated, onBotSaved, onBotDeleted }) {
  function handleNewBotClick() {
    openModal(<BotSettingsModal onSaved={onBotCreated} />);
  }

  function handleEditClick(bot) {
    openModal(<BotSettingsModal bot={bot} onSaved={onBotSaved} onDeleted={onBotDeleted} />);
  }

  const rows = bots.map(bot => (
    <SettingsRow key={bot.botId} icon={<span className={`settings-modal-bot-icon color-${bot.color}`}><BotAvatar name={bot.icon} /></span>} label={bot.name} description={bot.model}>
      <Button onClick={() => handleEditClick(bot)}>Edit</Button>
    </SettingsRow>
  ));

  let card = null;
  if (rows.length > 0) {
    card = <div className="settings-card">{rows}</div>;
  }

  return (
    <div className="settings-pane">
      <SettingsSection title="Bots" description="Each bot has its own model, memories, files and schedules.">
        {card}
        <div className="settings-form">
          <Button onClick={handleNewBotClick}><NewIcon />New bot</Button>
        </div>
      </SettingsSection>
    </div>
  );
}
