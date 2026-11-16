import Button from '../../commons/components/Button.jsx';
import Input from '../../commons/components/Input.jsx';
import SettingsSection from '../../commons/components/SettingsSection.jsx';
import Spinner from '../../commons/components/Spinner.jsx';
import { showToast } from '../../commons/components/Toast.jsx';
import ApiClient from '../../commons/http/ApiClient.js';

const { useState, useEffect } = React;

const MAX_LENGTH = 2000;

export default function AboutMePane() {
  const [aboutMe, setAboutMe] = useState("");
  const [isLoading, setIsLoading] = useState(true);
  const [isSaving, setIsSaving] = useState(false);

  useEffect(() => {
    ApiClient.getCurrentUser()
      .then(user => setAboutMe(user.aboutMe))
      .finally(() => setIsLoading(false));
  }, []);

  function handleSubmit(event) {
    event.preventDefault();
    setIsSaving(true);
    ApiClient.updateAboutMe({ aboutMe })
      .then(user => {
        setAboutMe(user.aboutMe);
        showToast("Saved");
      })
      .finally(() => setIsSaving(false));
  }

  let content = <Spinner />;
  if (!isLoading) {
    const length = [...aboutMe].length;
    let error = "";
    if (length > MAX_LENGTH) {
      error = `Keep it under ${MAX_LENGTH} characters.`;
    }

    content = (
      <form className="settings-form" onSubmit={handleSubmit}>
        <Input id="about-me" type="textarea" rows={8} placeholder={"I live in Singapore.\nTwo kids, 8 and 11.\nKeep answers short."} hint={`${length} / ${MAX_LENGTH} characters`} error={error} value={aboutMe} onChange={event => setAboutMe(event.target.value)} />
        <Button type="submit" variant="primary" isDisabled={isSaving || length > MAX_LENGTH}>Save</Button>
      </form>
    );
  }

  return (
    <div className="settings-pane">
      <SettingsSection title="About me" description="Facts every bot should know about you, added to each bot's instructions. Turn it off for a bot in its General settings.">
        {content}
      </SettingsSection>
    </div>
  );
}
