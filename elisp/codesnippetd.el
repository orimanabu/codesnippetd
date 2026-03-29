;;; codesnippetd.el --- Client for codesnippetd /pipe endpoint -*- lexical-binding: t -*-

;; Author: codesnippetd contributors
;; Keywords: tools

;;; Commentary:
;; Provides a command to POST text to the codesnippetd /pipe endpoint.
;; The text between mark and point (active region) is sent.

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
(defun codesnippetd-post-region ()
  "POST the active region to the codesnippetd /pipe endpoint.
Send the text between mark and point.  Signal an error if the region
is not active."
  (interactive)
  (unless (use-region-p)
    (user-error "codesnippetd: no active region"))
  (let* ((url (codesnippetd--pipe-url))
         (content (buffer-substring-no-properties (region-beginning) (region-end)))
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
