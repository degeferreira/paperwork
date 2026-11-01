# Go migration (in progress)

Scope: only go/, root go.mod/go.sum. Python and production untouched.
Read all Python modules and API/schema/PFM/settings/tasks/transcription/backup tests.
Compatibility invariants: retain raw historical payload TEXT bytes for proposal_hash;
UUID5 namespace 60500c63-b114-4bc1-acda-7b83d033ca97; schema user_version=1;
library_id/task_sequence never regenerated for existing libraries.
Temporary toolchain coordination: use /tmp/paperwork-go-toolchain, installation
serialized by /tmp/paperwork-go-install.lock. No production installation.
