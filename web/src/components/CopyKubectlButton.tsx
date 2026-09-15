import { useRef, useState } from "react";
import "./CopyKubectlButton.css";

type CopyKubectlButtonProps = {
  label: string;
  command: string;
  compact?: boolean;
  tone?: "default" | "danger";
};

export function CopyKubectlButton({
  label,
  command,
  compact = false,
  tone = "default",
}: CopyKubectlButtonProps) {
  const [copied, setCopied] = useState(false);
  const timer = useRef<number | null>(null);

  const copy = async () => {
    try {
      await navigator.clipboard.writeText(command);
    } catch {
      window.prompt("Copy kubectl command", command);
      return;
    }
    setCopied(true);
    if (timer.current != null) window.clearTimeout(timer.current);
    timer.current = window.setTimeout(() => setCopied(false), 1600);
  };

  return (
    <button
      type="button"
      className={[
        "copy-kubectl",
        compact ? "compact" : "",
        tone === "danger" ? "danger" : "",
      ]
        .filter(Boolean)
        .join(" ")}
      onClick={() => void copy()}
      title={command}
    >
      {copied ? "Copied" : label}
    </button>
  );
}
