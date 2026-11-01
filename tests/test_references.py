import sqlite3
import unittest
from paperwork.references import register,resolve
from paperwork.pfm import parse
class ReferenceTests(unittest.TestCase):
    def test_alias_permanent_and_personal_group_identity_distinct(self):
        db=sqlite3.connect(':memory:')
        try:
            register(db,'Silva24-Mem','users',123,'ABCD1234','Literature/Silva')
            register(db,'Silva24-Grupo','groups',123,'ABCD1234')
            with self.assertRaises(ValueError):register(db,'Silva24-Mem','groups',123,'ABCD1234')
            result=resolve(db,parse(['// 2026-10-31\n• Ƶ Silva24-Mem p.12\n• Ƶ Desconhecido'],'scan'))
            self.assertEqual(result[0]['identity']['library_type'],'users')
            self.assertEqual(result[0]['page'],12)
            self.assertIsNone(result[1]['identity'])
        finally:db.close()
