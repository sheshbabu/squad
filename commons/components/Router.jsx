const { useEffect, useState } = React;

export default function Router({ children }) {
  const [currentPath, setCurrentPath] = useState(window.location.pathname);

  useEffect(() => {
    function handleLocationChange() {
      setCurrentPath(window.location.pathname);
    }

    window.addEventListener("navigate", handleLocationChange);
    window.addEventListener("popstate", handleLocationChange);

    return () => {
      window.removeEventListener("navigate", handleLocationChange);
      window.removeEventListener("popstate", handleLocationChange);
    };
  }, []);

  const routes = React.Children.toArray(children);

  for (const route of routes) {
    const { path, component } = route.props;

    if (path.includes(":")) {
      const pathPattern = path.split("/").map(segment => {
        if (segment.startsWith(":")) {
          return `(?<${segment.slice(1)}>[^/]+)`;
        }
        return segment;
      }).join("\\/");

      const match = new RegExp(`^${pathPattern}\\/?$`).exec(currentPath);
      if (match !== null) {
        const params = {};
        for (const [key, value] of Object.entries(match.groups)) {
          params[key] = decodeURIComponent(value);
        }
        return React.createElement(component, params);
      }
    }

    if (currentPath === path || currentPath === path + "/") {
      return React.createElement(component, {});
    }
  }

  return null;
}
