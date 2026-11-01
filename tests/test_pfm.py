import unittest
from paperwork.pfm import parse

class PFMTests(unittest.TestCase):
    def test_three_days_and_cross_page(self):
        data=parse(['// 2026-10-29\n• A','> □ B\n//end\n// 2026-10-30\n• C','// 2026-10-31\n• D\n//end'],'scan')
        a,b,c,d=data['blocks']
        self.assertEqual(b['parent_id'],a['id'])
        self.assertEqual([a['context'],c['context'],d['context']],['Daily/2026-10-29','Daily/2026-10-30','Daily/2026-10-31'])
        self.assertEqual(data['review'],[])
    def test_mixed_tree_and_inheritance(self):
        data=parse(['// 2026-10-31\n• Leitura Ƶ Silva24-Mem p.12 @tese #memoria\n   • Argumento\n> ! Interpretação\n>> ? Evidência\n• Planejamento\n> □ Aula\n>> □ Slides'],'scan')
        a,b,c,d,e,f,g=data['blocks']
        self.assertEqual([b['parent_id'],c['parent_id'],d['parent_id']],[a['id'],a['id'],c['id']])
        self.assertEqual(d['references'],[{'alias':'Silva24-Mem','page':12}])
        self.assertEqual(g['parent_id'],f['id']); self.assertEqual(d['kind'],'question')
    def test_invalid_dates_and_pending_context(self):
        data=parse(['• Sem data\n// 2026-02-30\n• Inválido\n//end\n• Pendente'],'a')
        self.assertTrue(all(b['context'] is None for b in data['blocks']))
        self.assertIn('invalid-date',[r['reason'] for r in data['review']])
    def test_stable_ids(self):
        self.assertEqual(parse(['// 2026-10-31\n□ Fazer'],'a'),parse(['// 2026-10-31\n□ Fazer'],'a'))
    def test_missing_parent_and_literal(self):
        data=parse(['// 2026-10-31\n>> □ Órfão\n• Pai\n\\> literal\n>0 matemática'],'a')
        self.assertIsNone(data['blocks'][0]['parent_id'])
        self.assertIn('> literal',data['blocks'][1]['text'])
        self.assertEqual(len(data['blocks']),2)
    def test_long_note_and_continue(self):
        data=parse([':: Ensaio\n• Ideia\n::end\n// 2026-10-31\n//continue B01842\n• Segue'],'a')
        self.assertEqual(data['blocks'][0]['context'],'Page/Ensaio')
        self.assertEqual(data['blocks'][1]['continuation'],'B01842')
        self.assertEqual(data['review'][0]['reason'],'resolve-continuation')

if __name__=='__main__': unittest.main()

class FreePaperTests(unittest.TestCase):
    def test_assigned_date_does_not_change_transcription_or_coordinates(self):
        result=parse(['Texto livre sem marcações.\nContinuação.'],'scan','Daily/2026-10-31')
        self.assertEqual(result['review'],[])
        self.assertEqual(result['blocks'][0]['context'],'Daily/2026-10-31')
        self.assertEqual(result['blocks'][0]['line'],1)
        self.assertEqual(result['blocks'][0]['text'],'Texto livre sem marcações.\nContinuação.')
