import { closeModal } from '../../commons/components/Modal.jsx';
import SettingsSection from '../../commons/components/SettingsSection.jsx';
import showToast from '../../commons/components/Toast.jsx';
import Input from '../../commons/components/Input.jsx';
import Button from '../../commons/components/Button.jsx';
import Dropdown from '../../commons/components/Dropdown.jsx';
import SettingsRow from '../../commons/components/SettingsRow.jsx';
import Toggle from '../../commons/components/Toggle.jsx';
import BotAvatar, { BOT_ICONS, DEFAULT_BOT_ICON } from '../../commons/components/BotAvatar.jsx';
import { CheckIcon } from '../../commons/components/Icon.jsx';
import ApiClient from '../../commons/http/ApiClient.js';
import './BotForm.css';

const { useState, useEffect } = React;

const REASONING_EFFORT_LABELS = {
  none: "Off",
  minimal: "Minimal",
  low: "Low",
  medium: "Medium",
  high: "High",
  xhigh: "Extra high",
  max: "Max",
};

const COLORS = ["gray", "red", "orange", "yellow", "green", "teal", "blue", "purple", "pink"];

// A new bot needs every field before it can be created, while an existing bot edits its profile and its model in separate panes.
export default function BotForm({ bot, sections, onSaved, onDeleted }) {
  const hasProfile = sections.includes("profile");
  const hasModel = sections.includes("model");

  const [name, setName] = useState(bot?.name ?? "");
  const [color, setColor] = useState(bot?.color ?? "gray");
  const [icon, setIcon] = useState(bot?.icon || DEFAULT_BOT_ICON);
  const [systemPrompt, setSystemPrompt] = useState(bot?.systemPrompt ?? "");
  const [gateway, setGateway] = useState(bot?.gateway ?? "");
  const [model, setModel] = useState(bot?.model ?? "");
  const [provider, setProvider] = useState(bot?.provider ?? "");
  const [reasoningEffort, setReasoningEffort] = useState(bot?.reasoningEffort ?? "");
  const [usesAboutMe, setUsesAboutMe] = useState(bot?.usesAboutMe ?? true);
  const [gateways, setGateways] = useState([]);
  const [models, setModels] = useState([]);
  const [providers, setProviders] = useState([]);
  const [isSaving, setIsSaving] = useState(false);

  useEffect(() => {
    ApiClient.getGateways().then(names => {
      setGateways(names);
      if (gateway === "" && names.length > 0) {
        setGateway(names[0]);
      }
    });
  }, []);

  useEffect(() => {
    if (gateway === "") {
      return;
    }
    ApiClient.getModels(gateway).then(setModels);
  }, [gateway]);

  const isKnownModel = models.some(item => item.modelId === model);

  // A saved model the gateway no longer lists has no providers to fetch.
  useEffect(() => {
    if (!isKnownModel) {
      setProviders([]);
      return;
    }
    ApiClient.getProviders(gateway, model).then(setProviders);
  }, [gateway, model, isKnownModel]);

  function handleSubmit(event) {
    event.preventDefault();
    setIsSaving(true);

    const payload = { name, color, icon, systemPrompt, gateway, model, provider, reasoningEffort, usesAboutMe };
    const promise = bot === undefined ? ApiClient.createBot(payload) : ApiClient.updateBot(bot.botId, payload);

    // A new bot closes the modal and opens its chat, while an edit keeps the settings open like Zen's.
    promise
      .then(saved => {
        if (bot === undefined) {
          closeModal();
        } else {
          showToast("Saved");
        }
        onSaved(saved);
      })
      .finally(() => setIsSaving(false));
  }

  function handleDeleteClick() {
    if (!window.confirm(`Delete ${bot.name} with all its chats, files, memories and schedules?`)) {
      return;
    }
    ApiClient.deleteBot(bot.botId).then(() => {
      closeModal();
      onDeleted();
    });
  }

  function handleModelChange(modelId) {
    setModel(modelId);
    setProvider("");
    setReasoningEffort("");
  }

  const colorOptions = COLORS.map(item => {
    const isSelected = item === color;
    let checkIcon = null;
    if (isSelected) {
      checkIcon = <CheckIcon />;
    }

    return (
      <button key={item} type="button" title={item} className={`bot-form-color color-${item} ${isSelected ? "is-selected" : ""}`} onClick={() => setColor(item)}>
        {checkIcon}
      </button>
    );
  });

  const iconOptions = BOT_ICONS.map(item => (
    <button key={item.name} type="button" title={item.label} className={`bot-form-icon ${item.name === icon ? "is-selected" : ""}`} onClick={() => setIcon(item.name)}>
      <BotAvatar name={item.name} />
    </button>
  ));

  const gatewayOptions = gateways.map(item => ({ value: item, label: item }));

  const modelOptions = models.map(item => ({ value: item.modelId, label: item.name, description: item.modelId }));
  if (model !== "" && !isKnownModel) {
    modelOptions.unshift({ value: model, label: model });
  }

  const providerOptions = [{ value: "", label: "Any provider" }];
  for (const item of providers) {
    providerOptions.push({ value: item.providerId, label: item.name, description: item.providerId });
  }
  if (provider !== "" && !providers.some(item => item.providerId === provider)) {
    providerOptions.push({ value: provider, label: provider });
  }

  let reasoningDropdown = null;
  const selectedModel = models.find(item => item.modelId === model);
  if (selectedModel !== undefined && selectedModel.reasoningEfforts !== null && selectedModel.reasoningEfforts.length > 0) {
    let defaultLabel = "Default";
    if (selectedModel.defaultReasoningEffort !== "") {
      defaultLabel = `Default (${getReasoningEffortLabel(selectedModel.defaultReasoningEffort)})`;
    }
    const reasoningOptions = [{ value: "", label: defaultLabel }];
    for (const effort of selectedModel.reasoningEfforts) {
      reasoningOptions.push({ value: effort, label: getReasoningEffortLabel(effort) });
    }
    reasoningDropdown = <Dropdown id="bot-reasoning-effort" label="Thinking" hint="Less thinking is cheaper and faster, more helps with harder tasks." options={reasoningOptions} value={reasoningEffort} onChange={setReasoningEffort} />;
  }

  let profileSection = null;
  if (hasProfile) {
    profileSection = (
      <SettingsSection title="Profile">
        <Input id="bot-name" label="Name" value={name} onChange={event => setName(event.target.value)} />

        <div className="form-field-container">
          <label>Color</label>
          <div className="bot-form-colors">{colorOptions}</div>
        </div>

        <div className="form-field-container">
          <label>Icon</label>
          <div className={`bot-form-icons color-${color}`}>{iconOptions}</div>
        </div>

        <Input id="bot-system-prompt" label="Instructions" type="textarea" hint="Who the bot is and how it should behave." value={systemPrompt} onChange={event => setSystemPrompt(event.target.value)} />

        <div className="settings-card bot-form-about-me">
          <SettingsRow label="Use About me" description="Adds your About me from Settings to this bot's instructions.">
            <Toggle isEnabled={usesAboutMe} onChange={setUsesAboutMe} />
          </SettingsRow>
        </div>
      </SettingsSection>
    );
  }

  let modelSection = null;
  if (hasModel) {
    modelSection = (
      <SettingsSection title="Model">
        <Dropdown id="bot-gateway" label="Gateway" options={gatewayOptions} value={gateway} onChange={setGateway} />
        <Dropdown id="bot-model" label="Model" hint="Only models that support tool calls are listed." placeholder="Select a model" options={modelOptions} value={model} isSearchable onChange={handleModelChange} />
        <Dropdown id="bot-provider" label="Provider" options={providerOptions} value={provider} isSearchable onChange={setProvider} />
        {reasoningDropdown}
      </SettingsSection>
    );
  }

  let deleteSection = null;
  if (bot !== undefined && hasProfile) {
    deleteSection = (
      <SettingsSection title="Delete bot" description="Removes the bot with all its chats, files, memories and schedules.">
        <Button variant="danger" onClick={handleDeleteClick}>Delete bot</Button>
      </SettingsSection>
    );
  }

  return (
    <div className="settings-pane bot-form">
      <form className="settings-form" onSubmit={handleSubmit}>
        {profileSection}
        {modelSection}
        <Button type="submit" variant="primary" isDisabled={isSaving || name.trim() === "" || model.trim() === ""}>Save</Button>
      </form>
      {deleteSection}
    </div>
  );
}

function getReasoningEffortLabel(effort) {
  return REASONING_EFFORT_LABELS[effort] ?? effort;
}
