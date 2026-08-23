import os
import requests
import uuid
import xml.etree.ElementTree as ET
from typing import List, Dict, Optional

class CardDAVClient:
    def __init__(self):
        self.username = os.environ.get("MAILRU_USERNAME")
        self.password = os.environ.get("MAILRU_APP_PASS")
        self.host = "https://carddav.mail.ru"
        
        if not self.username or not self.password:
            raise ValueError("MAILRU_USERNAME and MAILRU_APP_PASS must be set for CardDAV.")
            
        self.auth = (self.username, self.password)

    def _discover_addressbook_url(self) -> str:
        # A simplified discovery. In a robust setup, we parse PROPFIND for current-user-principal,
        # then addressbook-home-set. For Mail.ru, it's typically predictable or standard.
        # Fallback to a standard path if discovery is complex.
        return f"{self.host}/addressbooks/"

    def search_contacts(self, query: str) -> List[Dict[str, str]]:
        """Search contacts using CardDAV REPORT."""
        # Note: Implementing a full REPORT query. For brevity and robustness in the agent,
        # we construct the XML payload.
        url = self._discover_addressbook_url()
        headers = {"Content-Type": "application/xml; charset=utf-8", "Depth": "1"}
        
        # addressbook-query payload
        body = f"""<?xml version="1.0" encoding="utf-8" ?>
            <c:addressbook-query xmlns:d="DAV:" xmlns:c="urn:ietf:params:xml:ns:carddav">
                <d:prop>
                    <d:getetag />
                    <c:address-data />
                </d:prop>
                <c:filter test="anyof">
                    <c:prop-filter name="FN">
                        <c:text-match collation="i;unicode-casemap" match-type="contains">{query}</c:text-match>
                    </c:prop-filter>
                    <c:prop-filter name="EMAIL">
                        <c:text-match collation="i;unicode-casemap" match-type="contains">{query}</c:text-match>
                    </c:prop-filter>
                </c:filter>
            </c:addressbook-query>"""
            
        res = requests.request("REPORT", url, auth=self.auth, headers=headers, data=body, timeout=10)
        
        if res.status_code not in (200, 207):
            return [{"error": f"Failed to search contacts: HTTP {res.status_code}"}]
            
        # Simplistic parse of VCards for the LLM
        contacts = []
        if "BEGIN:VCARD" in res.text:
            # We'd parse the vCards properly here.
            # Just slicing raw VCards for the AI to read.
            raw_vcards = res.text.split("BEGIN:VCARD")[1:]
            for vc in raw_vcards:
                contacts.append({"vcard": "BEGIN:VCARD" + vc[:vc.find("END:VCARD")+9]})
                
        return contacts if contacts else [{"info": "No contacts found matching query."}]

    def create_contact(self, name: str, email: str, phone: Optional[str] = None) -> bool:
        """Create a new contact (vCard)."""
        uid = str(uuid.uuid4())
        vcard = f"BEGIN:VCARD\r\nVERSION:3.0\r\nUID:{uid}\r\nFN:{name}\r\nEMAIL;TYPE=WORK,PREF:{email}\r\n"
        if phone:
            vcard += f"TEL;TYPE=CELL:{phone}\r\n"
        vcard += "END:VCARD\r\n"
        
        url = f"{self._discover_addressbook_url()}/{uid}.vcf"
        headers = {"Content-Type": "text/vcard; charset=utf-8", "If-None-Match": "*"}
        
        res = requests.put(url, auth=self.auth, headers=headers, data=vcard.encode('utf-8'), timeout=10)
        
        if res.status_code not in (200, 201, 204):
            raise RuntimeError(f"Failed to create contact: HTTP {res.status_code} - {res.text}")
        return True
