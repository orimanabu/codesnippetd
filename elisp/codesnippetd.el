;;; codesnippetd.el --- Client for codesnippetd /pipe endpoint -*- lexical-binding: t -*-

;; Author: codesnippetd contributors
;; Keywords: tools

;;; Commentary:
;; Provides a command to POST the current buffer's content to the
;; codesnippetd /pipe endpoint.

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
(defun codesnippetd-post-buffer ()
  "POST the current buffer's content to the codesnippetd /pipe endpoint."
  (interactive)
  (let* ((url (codesnippetd--pipe-url))
         (content (buffer-string))
         (url-request-method "POST")
         (url-request-extra-headers '(("Content-Type" . "application/octet-stream")))
         (url-request-data (encode-coding-string content 'utf-8)))
    (url-retrieve
     url
     (lambda (status)
       (if (plist-get status :error)
           (message "codesnippetd: POST failed: %s" (plist-get status :error))
         (message "codesnippetd: buffer posted to %s" url)))
     nil
     t)))

(provide 'codesnippetd)
;;; codesnippetd.el ends here
