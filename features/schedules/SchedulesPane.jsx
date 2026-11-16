import Button from '../../commons/components/Button.jsx';
import Spinner from '../../commons/components/Spinner.jsx';
import Toggle from '../../commons/components/Toggle.jsx';
import Link from '../../commons/components/Link.jsx';
import { closeModal } from '../../commons/components/Modal.jsx';
import { PencilIcon, TrashIcon, PlayIcon, HistoryIcon } from '../../commons/components/Icon.jsx';
import ApiClient from '../../commons/http/ApiClient.js';
import formatDate from '../../commons/utils/formatDate.js';
import navigateTo from '../../commons/utils/navigateTo.js';
import ScheduleForm from './ScheduleForm.jsx';
import './SchedulesPane.css';

const { useState, useEffect } = React;

const WEEKDAY_NAMES = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"];

export default function SchedulesPane({ botId }) {
  const [schedules, setSchedules] = useState([]);
  const [isLoading, setIsLoading] = useState(true);
  const [editingSchedule, setEditingSchedule] = useState(null);
  const [isCreating, setIsCreating] = useState(false);
  const [historyScheduleId, setHistoryScheduleId] = useState(null);
  const [runs, setRuns] = useState([]);

  useEffect(() => {
    ApiClient.getSchedules(botId)
      .then(setSchedules)
      .finally(() => setIsLoading(false));
  }, [botId]);

  function replaceSchedule(saved) {
    setSchedules(previous => {
      const others = previous.filter(schedule => schedule.scheduleId !== saved.scheduleId);
      return [...others, saved].sort((a, b) => a.name.localeCompare(b.name));
    });
  }

  function handleToggle(schedule, isEnabled) {
    ApiClient.updateSchedule(botId, schedule.scheduleId, { ...schedule, isEnabled }).then(replaceSchedule);
  }

  function handleRunClick(schedule) {
    ApiClient.runSchedule(botId, schedule.scheduleId).then(thread => {
      closeModal();
      navigateTo(`/bots/${botId}/threads/${thread.threadId}`);
    });
  }

  function handleHistoryClick(schedule) {
    if (historyScheduleId === schedule.scheduleId) {
      setHistoryScheduleId(null);
      return;
    }

    setRuns([]);
    setHistoryScheduleId(schedule.scheduleId);
    ApiClient.getScheduleThreads(botId, schedule.scheduleId).then(setRuns);
  }

  function handleRunLinkClick() {
    closeModal();
  }

  function handleDeleteClick(schedule) {
    if (!window.confirm(`Delete the schedule "${schedule.name}"?`)) {
      return;
    }
    ApiClient.deleteSchedule(botId, schedule.scheduleId)
      .then(() => setSchedules(previous => previous.filter(item => item.scheduleId !== schedule.scheduleId)));
  }

  function handleSaved(saved) {
    replaceSchedule(saved);
    setEditingSchedule(null);
    setIsCreating(false);
  }

  function handleCancel() {
    setEditingSchedule(null);
    setIsCreating(false);
  }

  if (isLoading) {
    return <Spinner />;
  }

  if (isCreating || editingSchedule !== null) {
    return <ScheduleForm botId={botId} schedule={editingSchedule} onSaved={handleSaved} onCancel={handleCancel} />;
  }

  const items = schedules.map(schedule => {
    let historyElement = null;
    if (historyScheduleId === schedule.scheduleId) {
      historyElement = <ScheduleRuns botId={botId} runs={runs} onLinkClick={handleRunLinkClick} />;
    }

    return (
      <div key={schedule.scheduleId} className="schedules-pane-entry">
        <div className="schedules-pane-item">
          <div className="schedules-pane-details">
            <div className="schedules-pane-name">{schedule.name}</div>
            <div className="schedules-pane-timing">{describeTiming(schedule)}</div>
          </div>
          <Toggle isEnabled={schedule.isEnabled} onChange={isEnabled => handleToggle(schedule, isEnabled)} />
          <Button variant="ghost" onClick={() => handleRunClick(schedule)} title="Run now"><PlayIcon /></Button>
          <Button variant="ghost" onClick={() => handleHistoryClick(schedule)} title="History"><HistoryIcon /></Button>
          <Button variant="ghost" onClick={() => setEditingSchedule(schedule)} title="Edit"><PencilIcon /></Button>
          <Button variant="ghost" onClick={() => handleDeleteClick(schedule)} title="Delete"><TrashIcon /></Button>
        </div>
        {historyElement}
      </div>
    );
  });

  let emptyElement = null;
  if (schedules.length === 0) {
    emptyElement = <div className="schedules-pane-empty">No schedules yet. A schedule sends the bot a task at a set time, and each run starts a new chat.</div>;
  }

  return (
    <div className="schedules-pane">
      {emptyElement}
      {items}
      <div className="schedules-pane-footer">
        <Button onClick={() => setIsCreating(true)}>Add schedule</Button>
      </div>
    </div>
  );
}

function ScheduleRuns({ botId, runs, onLinkClick }) {
  if (runs.length === 0) {
    return <div className="schedules-pane-runs is-empty">No runs yet</div>;
  }

  const items = runs.map(run => {
    const status = getRunStatus(run);

    return (
      <div key={run.threadId} className="schedules-pane-run">
        <Link className="schedules-pane-run-title" to={`/bots/${botId}/threads/${run.threadId}`} onClick={onLinkClick}>{run.title}</Link>
        <span className="schedules-pane-run-meta">
          <span className={`schedules-pane-run-status ${status.className}`}>{status.label}</span>
          {formatDate(new Date(run.createdAt))}
        </span>
      </div>
    );
  });

  return <div className="schedules-pane-runs">{items}</div>;
}

function getRunStatus(run) {
  if (run.status === "working") {
    return { label: "Running", className: "is-working" };
  }
  if (run.status === "failed") {
    return { label: "Failed", className: "is-failed" };
  }
  return { label: "Done", className: "is-done" };
}

function describeTiming(schedule) {
  let days = "Every day";
  if (schedule.frequency === "weekly") {
    days = schedule.weekdays.map(weekday => WEEKDAY_NAMES[weekday]).join(", ");
  }

  let nextRun = "";
  if (schedule.isEnabled) {
    nextRun = ` · next ${new Date(schedule.nextRunAt).toLocaleString(undefined, { weekday: "short", hour: "2-digit", minute: "2-digit" })}`;
  }

  return `${days} at ${schedule.timeOfDay}${nextRun}`;
}
