;;; codesnippetd.el --- Client for codesnippetd /pipe endpoint -*- lexical-binding: t -*-

;; Author: codesnippetd contributors
;; Keywords: tools

;;; Commentary:
;; Provides a command to POST text to the codesnippetd /pipe endpoint.
;; If the region is active, the selected text is sent; otherwise the
;; most recent kill-ring entry is sent.

;;; Code:

(require 'url)

(defgroup codesnippetd nil
  "Client for the codesnippetd REST API server."
  :group 'tools
  :prefix "codesnippetd-")

(defcustom codesnippetd-host "localhost"
  "Hostname of the codesnippetd server."
  :type 'string
  :group 'codesnippetd)

(defcustom codesnippetd-port 8999
  "Port number of the codesnippetd server."
  :type 'integer
  :group 'codesnippetd)

(defun codesnippetd--pipe-url ()
  "Return the URL for the /pipe endpoint."
  (format "http://%s:%d/pipe" codesnippetd-host codesnippetd-port))

;;;###autoload
(defun codesnippetd-post-region-or-kill ()
  "POST to the codesnippetd /pipe endpoint.
If the region is active, send the text between mark and point.
Otherwise, send the contents of the kill ring (most recent yank)."
  (interactive)
  (let* ((url (codesnippetd--pipe-url))
         (content (if (use-region-p)
                      (buffer-substring-no-properties (region-beginning) (region-end))
                    (or (car kill-ring)
                        (user-error "codesnippetd: kill ring is empty"))))
         (url-request-method "POST")
         (url-request-extra-headers '(("Content-Type" . "application/octet-stream")))
         (url-request-data (encode-coding-string content 'utf-8)))
    (url-retrieve
     url
     (lambda (status)
       (if (plist-get status :error)
           (message "codesnippetd: POST failed: %s" (plist-get status :error))
         (message "codesnippetd: sent to %s" url)))
     nil
     t)))

(provide 'codesnippetd)
;;; codesnippetd.el ends here
