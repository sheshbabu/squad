import SettingsSection from '../../commons/components/SettingsSection.jsx';
import SegmentedControl from '../../commons/components/SegmentedControl.jsx';
import Spinner from '../../commons/components/Spinner.jsx';
import Link from '../../commons/components/Link.jsx';
import { closeModal } from '../../commons/components/Modal.jsx';
import ApiClient from '../../commons/http/ApiClient.js';
import './CostPane.css';

const { useState, useEffect } = React;

const RANGE_OPTIONS = [
  { value: "7d", label: "7 days" },
  { value: "30d", label: "30 days" },
  { value: "all", label: "All time" },
];

export default function CostPane({ botId }) {
  const [range, setRange] = useState("30d");
  const [cost, setCost] = useState(null);

  useEffect(() => {
    ApiClient.getBotCost(botId, range).then(setCost);
  }, [botId, range]);

  // The previous range stays on screen while the next one loads, so switching doesn't flash a spinner.
  let summaryContent = <Spinner />;
  let detailSections = null;
  if (cost !== null && cost.totalCost === 0) {
    summaryContent = <div className="cost-pane-empty">No cost in this range.</div>;
  } else if (cost !== null) {
    summaryContent = (
      <>
        <CostStats cost={cost} />
        <CostChart buckets={cost.buckets} isMonthly={range === "all"} />
      </>
    );
    detailSections = (
      <>
        <SettingsSection title="Most expensive chats">
          <TopChats botId={botId} chats={cost.topChats} />
        </SettingsSection>
        <SettingsSection title="By model">
          <ModelCosts models={cost.models} />
        </SettingsSection>
      </>
    );
  }

  return (
    <div className="settings-pane cost-pane">
      <SettingsSection title="Cost" description="What this bot spent on replies. Chat titles aren't counted, so totals read slightly low.">
        <SegmentedControl options={RANGE_OPTIONS} value={range} onChange={setRange} />
        {summaryContent}
      </SettingsSection>
      {detailSections}
    </div>
  );
}

function CostStats({ cost }) {
  return (
    <div className="cost-pane-stats">
      <div className="cost-pane-stat">
        <div className="cost-pane-stat-value">{formatCost(cost.totalCost)}</div>
        <div className="cost-pane-stat-label">Total</div>
      </div>
      <div className="cost-pane-stat">
        <div className="cost-pane-stat-value">{cost.chatCount.toLocaleString()}</div>
        <div className="cost-pane-stat-label">Chats</div>
      </div>
      <div className="cost-pane-stat">
        <div className="cost-pane-stat-value">{formatCost(cost.totalCost / cost.chatCount)}</div>
        <div className="cost-pane-stat-label">Per chat</div>
      </div>
    </div>
  );
}

function CostChart({ buckets, isMonthly }) {
  const maxCost = Math.max(...buckets.map(bucket => bucket.chatCost + bucket.scheduledCost));

  const bars = buckets.map(bucket => {
    const label = formatBucketLabel(bucket.start, isMonthly);
    const total = bucket.chatCost + bucket.scheduledCost;
    const title = `${label}: ${formatCost(total)} (chats ${formatCost(bucket.chatCost)}, scheduled ${formatCost(bucket.scheduledCost)})`;

    const segments = [];
    if (bucket.scheduledCost > 0) {
      segments.push(<div key="scheduled" className="cost-pane-bar-segment is-scheduled" style={{ height: `${bucket.scheduledCost / maxCost * 100}%` }} />);
    }
    if (bucket.chatCost > 0) {
      segments.push(<div key="chat" className="cost-pane-bar-segment is-chat" style={{ height: `${bucket.chatCost / maxCost * 100}%` }} />);
    }

    return <div key={bucket.start} className="cost-pane-bar" title={title}>{segments}</div>;
  });

  return (
    <div className="cost-pane-chart">
      <div className="cost-pane-chart-max">{formatCost(maxCost)}</div>
      <div className="cost-pane-bars">{bars}</div>
      <div className="cost-pane-axis">
        <span>{formatBucketLabel(buckets[0].start, isMonthly)}</span>
        <span>{formatBucketLabel(buckets[buckets.length - 1].start, isMonthly)}</span>
      </div>
      <div className="cost-pane-legend">
        <span className="cost-pane-legend-item"><span className="cost-pane-swatch is-chat" />Chats</span>
        <span className="cost-pane-legend-item"><span className="cost-pane-swatch is-scheduled" />Scheduled</span>
      </div>
    </div>
  );
}

function TopChats({ botId, chats }) {
  const rows = chats.map(chat => (
    <Link key={chat.threadId} to={`/bots/${botId}/threads/${chat.threadId}`} className="cost-pane-row" onClick={closeModal}>
      <span className="cost-pane-row-name">{chat.title}</span>
      <span className="cost-pane-row-cost">{formatCost(chat.cost)}</span>
    </Link>
  ));

  return <div className="cost-pane-rows">{rows}</div>;
}

function ModelCosts({ models }) {
  const rows = models.map(item => (
    <div key={item.model} className="cost-pane-row">
      <span className="cost-pane-row-name">{item.model}</span>
      <span className="cost-pane-row-cost">{formatCost(item.cost)}</span>
    </div>
  ));

  return <div className="cost-pane-rows">{rows}</div>;
}

// Single replies often cost fractions of a cent, which two decimals would round to $0.00.
function formatCost(value) {
  if (value >= 1) {
    return `$${value.toFixed(2)}`;
  }
  return `$${value.toFixed(4)}`;
}

function formatBucketLabel(start, isMonthly) {
  if (isMonthly) {
    return new Date(start).toLocaleDateString(undefined, { month: "short", year: "numeric" });
  }
  return new Date(start).toLocaleDateString(undefined, { month: "short", day: "numeric" });
}
