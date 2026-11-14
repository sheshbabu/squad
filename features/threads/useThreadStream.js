const { useEffect, useRef } = React;

// EventSource reconnects on its own after drops, so there is no retry logic here.
export default function useThreadStream(threadId, handlers) {
  const handlersRef = useRef(handlers);
  handlersRef.current = handlers;

  useEffect(() => {
    if (threadId === undefined) {
      return;
    }

    const source = new EventSource(`/api/threads/${threadId}/events/`);

    source.onopen = () => handlersRef.current.onOpen();

    function listen(type, handlerName) {
      source.addEventListener(type, event => {
        handlersRef.current[handlerName](JSON.parse(event.data));
      });
    }

    listen("delta", "onDelta");
    listen("reset", "onReset");
    listen("message", "onMessage");
    listen("status", "onStatus");
    listen("thread", "onThread");

    return () => source.close();
  }, [threadId]);
}
