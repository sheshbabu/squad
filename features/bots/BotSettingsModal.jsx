import SidePaneModal from '../../commons/components/SidePaneModal.jsx';
import SettingsSection from '../../commons/components/SettingsSection.jsx';
import { SettingsIcon, BrainCircuitIcon, BotIcon, ClockIcon, ToolIcon, WebhookIcon } from '../../commons/components/Icon.jsx';
import BotForm from './BotForm.jsx';
import BotToolsPane from './BotToolsPane.jsx';
import BotWebhookPane from './BotWebhookPane.jsx';
import MemoriesPane from '../memories/MemoriesPane.jsx';
import SchedulesPane from '../schedules/SchedulesPane.jsx';

const { useState } = React;

export default function BotSettingsModal({ bot, onSaved, onDeleted }) {
  // Each pane saves the whole bot, so both start from the latest save and one pane can't undo the other's changes.
  const [currentBot, setCurrentBot] = useState(bot);

  function handleSaved(saved) {
    setCurrentBot(saved);
    onSaved(saved);
  }

  if (currentBot === undefined) {
    const newBotTab = { id: "general", label: "General", icon: <SettingsIcon />, content: <BotForm sections={["profile", "model"]} onSaved={onSaved} /> };
    return <SidePaneModal title="New bot" tabGroups={[[newBotTab]]} />;
  }

  const tabGroups = [
    [
      { id: "general", label: "General", icon: <SettingsIcon />, content: <BotForm bot={currentBot} sections={["profile"]} onSaved={handleSaved} onDeleted={onDeleted} /> },
      { id: "model", label: "Model", icon: <BotIcon />, content: <BotForm bot={currentBot} sections={["model"]} onSaved={handleSaved} /> },
      { id: "tools", label: "Tools", icon: <ToolIcon />, content: <BotToolsPane bot={currentBot} onSaved={handleSaved} /> },
      { id: "webhook", label: "Webhook", icon: <WebhookIcon />, content: <BotWebhookPane bot={currentBot} onSaved={handleSaved} /> },
    ],
    [
      {
        id: "memories",
        label: "Memories",
        icon: <BrainCircuitIcon />,
        content: (
          <div className="settings-pane">
            <SettingsSection title="Memories" description="Facts the bot saved to remember across chats.">
              <MemoriesPane botId={currentBot.botId} />
            </SettingsSection>
          </div>
        )
      },
      {
        id: "schedules",
        label: "Schedules",
        icon: <ClockIcon />,
        content: (
          <div className="settings-pane">
            <SettingsSection title="Schedules" description="Prompts the bot runs on its own, each in a new chat.">
              <SchedulesPane botId={currentBot.botId} />
            </SettingsSection>
          </div>
        )
      },
    ],
  ];

  return <SidePaneModal title={currentBot.name} tabGroups={tabGroups} />;
}
