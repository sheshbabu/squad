import Button from '../../commons/components/Button.jsx';
import Input from '../../commons/components/Input.jsx';
import SegmentedControl from '../../commons/components/SegmentedControl.jsx';
import ApiClient from '../../commons/http/ApiClient.js';

const { useState } = React;

const FREQUENCY_OPTIONS = [
  { value: "daily", label: "Daily" },
  { value: "weekly", label: "Weekly" },
];

const WEEKDAYS = [
  { value: 1, label: "Mon" },
  { value: 2, label: "Tue" },
  { value: 3, label: "Wed" },
  { value: 4, label: "Thu" },
  { value: 5, label: "Fri" },
  { value: 6, label: "Sat" },
  { value: 0, label: "Sun" },
];

export default function ScheduleForm({ botId, schedule, onSaved, onCancel }) {
  const [name, setName] = useState(schedule?.name ?? "");
  const [prompt, setPrompt] = useState(schedule?.prompt ?? "");
  const [frequency, setFrequency] = useState(schedule?.frequency ?? "daily");
  const [timeOfDay, setTimeOfDay] = useState(schedule?.timeOfDay ?? "08:00");
  const [weekdays, setWeekdays] = useState(schedule?.weekdays ?? [1, 2, 3, 4, 5]);
  const [isSaving, setIsSaving] = useState(false);

  function handleWeekdayChange(weekday) {
    setWeekdays(previous => {
      if (previous.includes(weekday)) {
        return previous.filter(item => item !== weekday);
      }
      return [...previous, weekday].sort();
    });
  }

  function handleSaveClick() {
    setIsSaving(true);

    const payload = { name, prompt, frequency, timeOfDay, weekdays, isEnabled: schedule?.isEnabled ?? true };
    let promise = ApiClient.createSchedule(botId, payload);
    if (schedule !== null) {
      promise = ApiClient.updateSchedule(botId, schedule.scheduleId, payload);
    }

    promise
      .then(onSaved)
      .finally(() => setIsSaving(false));
  }

  let weekdaysElement = null;
  if (frequency === "weekly") {
    const checkboxes = WEEKDAYS.map(weekday => (
      <label key={weekday.value} className="schedules-pane-weekday">
        <input type="checkbox" checked={weekdays.includes(weekday.value)} onChange={() => handleWeekdayChange(weekday.value)} />
        {weekday.label}
      </label>
    ));
    weekdaysElement = <div className="schedules-pane-weekdays">{checkboxes}</div>;
  }

  return (
    <div className="schedules-pane is-form">
      <Input id="schedule-name" label="Name" placeholder="Morning briefing" value={name} onChange={event => setName(event.target.value)} />
      <Input id="schedule-prompt" label="Task" type="textarea" placeholder="Summarise the top Hacker News stories from the last day." value={prompt} onChange={event => setPrompt(event.target.value)} />

      <div className="form-field-container">
        <label>When</label>
        <div className="schedules-pane-when">
          <SegmentedControl options={FREQUENCY_OPTIONS} value={frequency} onChange={setFrequency} />
          <input type="time" className="schedules-pane-time" value={timeOfDay} onChange={event => setTimeOfDay(event.target.value)} />
        </div>
        {weekdaysElement}
      </div>

      <div className="schedules-pane-footer">
        <Button onClick={onCancel}>Cancel</Button>
        <Button variant="primary" onClick={handleSaveClick} isDisabled={isSaving || name.trim() === "" || prompt.trim() === ""}>Save</Button>
      </div>
    </div>
  );
}
