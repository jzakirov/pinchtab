import { useState, useCallback, useEffect, useRef } from "react";
import { useParams, useSearchParams } from "react-router-dom";
import ScreencastTile from "../components/screencast/ScreencastTile";
import type { ServerMessage } from "../components/screencast/ScreencastTile";

type Tab = { id: string; url: string; title: string };

export default function LiveViewerPage() {
  const { token } = useParams<{ token: string }>();
  const [searchParams] = useSearchParams();

  const quality = Number(searchParams.get("quality") || "60");
  const maxWidth = Number(searchParams.get("maxWidth") || "1280");
  const fps = Number(searchParams.get("fps") || "15");

  const [tabs, setTabs] = useState<Tab[]>([]);
  const [currentTabId, setCurrentTabId] = useState("");
  const [pageUrl, setPageUrl] = useState("");
  const [pageTitle, setPageTitle] = useState("Browser");
  const sendRef = useRef<((msg: Record<string, unknown>) => void) | null>(null);

  const proto = location.protocol === "https:" ? "wss:" : "ws:";
  const wsUrl = `${proto}//${location.host}/live/${token}/screencast`;

  useEffect(() => { document.title = pageTitle || "Browser"; }, [pageTitle]);

  const onServerMessage = useCallback((msg: ServerMessage) => {
    const data = msg.data as Record<string, string> | Tab[] | undefined;
    if (msg.type === "urlChanged" && data && !Array.isArray(data)) {
      if (data.url) setPageUrl(data.url);
      if (data.title) setPageTitle(data.title);
    } else if (msg.type === "tabs" && Array.isArray(data)) {
      setTabs(data);
      setCurrentTabId((prev) => prev || (data.length > 0 ? data[0].id : ""));
    }
  }, []);

  const onReady = useCallback((send: (msg: Record<string, unknown>) => void) => {
    sendRef.current = send;
    send({ type: "getUrl" });
    send({ type: "getTabs" });
  }, []);

  const send = (type: string, url?: string) => {
    sendRef.current?.(url ? { type, url } : { type });
  };

  return (
    <div className="flex h-screen flex-col bg-[#202124] text-[#e8eaed]">
      {/* Tab bar */}
      {tabs.length > 1 && (
        <div className="flex items-end gap-px overflow-x-auto bg-[#202124] px-2 pt-1"
             style={{ scrollbarWidth: "none" }}>
          {tabs.map((t) => (
            <button
              key={t.id}
              onClick={() => setCurrentTabId(t.id)}
              className={`max-w-[200px] truncate rounded-t-lg px-3 py-1.5 text-xs select-none ${
                t.id === currentTabId
                  ? "bg-[#292a2d] text-[#e8eaed]"
                  : "text-[#9aa0a6] hover:bg-[#35363a]"
              }`}
            >
              {t.title || t.url || "New Tab"}
            </button>
          ))}
        </div>
      )}

      {/* Toolbar */}
      <div className="flex items-center gap-1 border-b border-[#3c4043] bg-[#292a2d] px-2 py-1">
        <NavBtn title="Back" onClick={() => send("back")}>
          <svg viewBox="0 0 24 24" className="h-4 w-4 fill-current"><path d="M20 11H7.83l5.59-5.59L12 4l-8 8 8 8 1.41-1.41L7.83 13H20v-2z" /></svg>
        </NavBtn>
        <NavBtn title="Forward" onClick={() => send("forward")}>
          <svg viewBox="0 0 24 24" className="h-4 w-4 fill-current"><path d="M12 4l-1.41 1.41L16.17 11H4v2h12.17l-5.58 5.59L12 20l8-8z" /></svg>
        </NavBtn>
        <NavBtn title="Reload" onClick={() => send("reload")}>
          <svg viewBox="0 0 24 24" className="h-4 w-4 fill-current"><path d="M17.65 6.35A7.958 7.958 0 0012 4c-4.42 0-7.99 3.58-7.99 8s3.57 8 7.99 8c3.73 0 6.84-2.55 7.73-6h-2.08A5.99 5.99 0 0112 18c-3.31 0-6-2.69-6-6s2.69-6 6-6c1.66 0 3.14.69 4.22 1.78L13 11h7V4l-2.35 2.35z" /></svg>
        </NavBtn>
        <input
          type="text"
          value={pageUrl}
          onChange={(e) => setPageUrl(e.target.value)}
          onKeyDown={(e) => {
            if (e.key !== "Enter") return;
            let val = pageUrl.trim();
            if (!val) return;
            if (!/^https?:\/\//i.test(val)) {
              val = /^[\w-]+\.[\w.-]+/.test(val)
                ? "https://" + val
                : "https://www.google.com/search?q=" + encodeURIComponent(val);
            }
            setPageUrl(val);
            send("navigate", val);
          }}
          onFocus={(e) => e.target.select()}
          placeholder="Search or enter URL"
          spellCheck={false}
          autoComplete="off"
          className="min-w-0 flex-1 rounded-full border border-[#3c4043] bg-[#202124] px-3.5 py-1 text-[13px] text-[#e8eaed] outline-none focus:border-[#8ab4f8]"
        />
      </div>

      {/* Screencast */}
      <div className="min-h-0 flex-1">
        <ScreencastTile
          instanceId=""
          tabId={currentTabId}
          label=""
          url={pageUrl}
          quality={quality}
          maxWidth={maxWidth}
          fps={fps}
          showTitle={false}
          wsUrlOverride={wsUrl}
          interactive
          onServerMessage={onServerMessage}
          onReady={onReady}
        />
      </div>
    </div>
  );
}

function NavBtn({ title, onClick, children }: { title: string; onClick: () => void; children: React.ReactNode }) {
  return (
    <button onClick={onClick} title={title}
      className="flex h-7 w-7 items-center justify-center rounded-full text-[#9aa0a6] hover:bg-[#35363a] hover:text-[#e8eaed]">
      {children}
    </button>
  );
}
