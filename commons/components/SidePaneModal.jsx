import { ModalBackdrop, ModalContainer, ModalHeader, closeModal } from './Modal.jsx';
import './SidePaneModal.css';

const { useState } = React;

// Tabs are grouped so related panes sit together, with a divider between groups, as in Zen's settings.
export default function SidePaneModal({ title, tabGroups, initialTabId }) {
  const tabs = tabGroups.flat();
  const [activeTabId, setActiveTabId] = useState(initialTabId ?? tabs[0].id);

  const sidebar = tabGroups.map((group, index) => {
    const groupTabs = group.map(tab => (
      <div key={tab.id} className={`side-pane-modal-tab ${activeTabId === tab.id ? "is-active" : ""}`} onClick={() => setActiveTabId(tab.id)}>
        {tab.icon}
        {tab.label}
      </div>
    ));

    return <div key={index} className="side-pane-modal-group">{groupTabs}</div>;
  });

  const activeTab = tabs.find(tab => tab.id === activeTabId) ?? tabs[0];

  return (
    <ModalBackdrop onClose={closeModal}>
      <ModalContainer className="side-pane-modal">
        <ModalHeader title={title} onClose={closeModal} />
        <div className="side-pane-modal-content">
          <div className="side-pane-modal-sidebar">{sidebar}</div>
          <div className="side-pane-modal-main">{activeTab.content}</div>
        </div>
      </ModalContainer>
    </ModalBackdrop>
  );
}
