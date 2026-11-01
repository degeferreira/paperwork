import unittest
from paperwork.pfm import parse
from paperwork.markdown import render
class MarkdownTests(unittest.TestCase):
    def test_additive_markdown_preserves_ids_and_no_relay_checkbox(self):
        proposal=parse(['// 2026-10-31\n□ Fazer ${evil()}\n> • Filho'],'scan')
        notes=render(proposal,'a'*64);text=notes['Daily/2026-10-31']
        self.assertIn('paperwork-block:'+proposal['blocks'][0]['id'],text)
        self.assertNotIn('${evil()',text)
        self.assertNotIn('- [ ]',text)
        self.assertIn('\n  - Filho',text)
    def test_unreviewed_cannot_be_incorporated(self):
        with self.assertRaises(ValueError):render(parse(['• Sem data'],'scan'),'a'*64)

class LongNoteTests(unittest.TestCase):
    def test_long_note_paragraphs_and_safe_line_breaks(self):
        proposal=parse([':: Ensaio\nPrimeiro parágrafo.\nContinuação.\n\nSegundo parágrafo.\n::end'],'scan')
        self.assertEqual(proposal['review'],[])
        self.assertEqual(len(proposal['blocks']),2)
        text=render(proposal,'a'*64)['Paperwork/Pages/Ensaio']
        self.assertIn('Primeiro parágrafo.<br>Continuação.',text)
        self.assertFalse(text.startswith('- '))

class UntrustedTextTests(unittest.TestCase):
    def test_imported_fences_images_and_links_stay_literal(self):
        from paperwork.markdown import literal
        unsafe='~~~space-lua\nevil()\n~~~\n![beacon](https://example.invalid/pixel)\n[run](javascript:evil())\n${evil()}'
        safe=literal(unsafe)
        self.assertNotIn('~~~',safe)
        self.assertNotIn('![beacon]',safe)
        self.assertNotIn('[run](',safe)
        self.assertNotIn('${evil()}',safe)
