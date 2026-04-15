;;; codesnippetd.el --- Client for codesnippetd /pipe endpoint -*- lexical-binding: t -*-

;; Author: codesnippetd contributors
;; Keywords: tools

;;; Commentary:
;; Provides a command to POST text to the codesnippetd /pipe endpoint.
;; The active region (mark to point) is sent as a JSON payload containing
;; the file path, start/end line numbers, and the selected code.

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

(defun codesnippetd--git-relative-path (fullpath)
  "Return FULLPATH relative to the nearest git root, or FULLPATH if not found.
Walk up the directory tree from FULLPATH looking for a .git directory.
If found, return the path portion after the git root.  If not found,
return FULLPATH unchanged."
  (let ((dir (file-name-directory fullpath)))
    (catch 'done
      (while t
        (if (file-directory-p (expand-file-name ".git" dir))
            (throw 'done (file-relative-name fullpath dir))
          (let ((parent (file-name-directory (directory-file-name dir))))
            (if (string= parent dir)
                (throw 'done fullpath)
              (setq dir parent)))))))))

;;;###autoload
(defun codesnippetd-post-region ()
  "POST the active region to the codesnippetd /pipe endpoint as JSON.
Send a JSON object with the file path, start/end line numbers, and the
selected code.  Signal an error if the region is not active."
  (interactive)
  (unless (use-region-p)
    (user-error "codesnippetd: no active region"))
  (let* ((url (codesnippetd--pipe-url))
         (code (buffer-substring-no-properties (region-beginning) (region-end)))
         (path (let ((f (buffer-file-name)))
                 (if f (codesnippetd--git-relative-path f) "")))
         (start (line-number-at-pos (region-beginning)))
         (end (line-number-at-pos (region-end)))
         (payload (json-encode `(("name"  . "")
                                 ("path"  . ,path)
                                 ("start" . ,start)
                                 ("end"   . ,end)
                                 ("code"  . ,code))))
         (url-request-method "POST")
         (url-request-extra-headers '(("Content-Type" . "application/json")))
         (url-request-data (encode-coding-string payload 'utf-8)))
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
