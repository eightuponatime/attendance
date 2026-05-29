import { AlertTriangle, X } from "lucide-react";
import { useI18n } from "../i18n/i18n";

export type ConfirmDialogTone = "danger" | "neutral";

type ConfirmDialogProps = {
  open: boolean;
  title: string;
  text: string;
  confirmText: string;
  cancelText?: string;
  tone?: ConfirmDialogTone;
  onConfirm: () => void;
  onCancel: () => void;
};

export function ConfirmDialog({
  open,
  title,
  text,
  confirmText,
  cancelText,
  tone = "neutral",
  onConfirm,
  onCancel,
}: ConfirmDialogProps) {
  const { t } = useI18n();
  if (!open) {
    return null;
  }

  return (
    <div className="confirm-backdrop" role="presentation" onMouseDown={onCancel}>
      <section
        className="confirm-dialog"
        role="dialog"
        aria-modal="true"
        aria-labelledby="confirm-dialog-title"
        onMouseDown={(event) => event.stopPropagation()}
      >
        <button className="confirm-close" type="button" aria-label={t("confirm.close")} onClick={onCancel}>
          <X size={19} />
        </button>

        <div className={`confirm-icon confirm-icon-${tone}`}>
          <AlertTriangle size={22} />
        </div>

        <div className="confirm-copy">
          <h2 id="confirm-dialog-title">{title}</h2>
          <p>{text}</p>
        </div>

        <div className="confirm-actions">
          <button className="confirm-cancel" type="button" onClick={onCancel}>
            {cancelText ?? t("confirm.cancel")}
          </button>
          <button className={`confirm-submit confirm-submit-${tone}`} type="button" onClick={onConfirm}>
            {confirmText}
          </button>
        </div>
      </section>
    </div>
  );
}
