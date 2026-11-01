import json
import os
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch
from paperwork.settings import configuration
class SharedConnectionTests(unittest.TestCase):
    def test_only_effective_settings_and_active_credentials_are_adopted(self):
        with tempfile.TemporaryDirectory() as folder:
            root=Path(folder);settings=root/'relay';settings.mkdir()
            original={'silverbullet_url':'http://sb','vikunja':{'url':'http://vk'},'timezone':'UTC','zotero':{'user_id':123}}
            source=root/'original.json';source.write_text(json.dumps(original))
            effective={**original,'timezone':'America/Sao_Paulo'}
            (settings/'effective.json').write_text(json.dumps({'connections':effective}))
            (settings/'integrations.json').write_text(json.dumps({**original,'timezone':'Europe/Paris'}))
            (settings/'secrets').mkdir();(settings/'secrets/vikunja_token').write_text('synthetic-only')
            with patch.dict(os.environ,{'PAPERWORK_CONFIG':str(root/'paperwork.json'),'PAPERWORK_RELAY_CONFIG':str(source),'PAPERWORK_RELAY_SETTINGS':str(settings)}):
                (settings/'secrets/silverbullet_token').write_text('synthetic-test-token')
                from paperwork.space import SpaceConnection
                connection=SpaceConnection()
                self.assertEqual(connection.url,'http://sb')
                self.assertEqual(connection.token_file,settings/'secrets/silverbullet_token')
                config=configuration()
                self.assertEqual(config['timezone'],'America/Sao_Paulo')
                self.assertTrue(config['__shared'])
                self.assertEqual(config['vikunja']['token_file'],str(settings/'secrets/vikunja_token'))
                self.assertNotIn('synthetic-only',json.dumps(config))
