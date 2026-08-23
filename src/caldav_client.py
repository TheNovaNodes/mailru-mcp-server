import os
import requests
import uuid
import datetime
from typing import List, Dict

class CalDAVClient:
    def __init__(self):
        self.username = os.environ.get("MAILRU_USERNAME")
        self.password = os.environ.get("MAILRU_APP_PASS")
        self.host = "https://caldav.mail.ru"
        
        if not self.username or not self.password:
            raise ValueError("MAILRU_USERNAME and MAILRU_APP_PASS must be set for CalDAV.")
            
        self.auth = (self.username, self.password)

    def _discover_calendar_url(self) -> str:
        # Fallback to standard calendar path
        return f"{self.host}/calendars/"

    def list_events(self, days_ahead: int = 7) -> List[str]:
        """Fetch upcoming calendar events using CalDAV REPORT."""
        url = self._discover_calendar_url()
        headers = {"Content-Type": "application/xml; charset=utf-8", "Depth": "1"}
        
        now = datetime.datetime.utcnow()
        end = now + datetime.timedelta(days=days_ahead)
        
        start_str = now.strftime("%Y%m%dT%H%M%SZ")
        end_str = end.strftime("%Y%m%dT%H%M%SZ")
        
        # calendar-query payload
        body = f"""<?xml version="1.0" encoding="utf-8" ?>
            <c:calendar-query xmlns:d="DAV:" xmlns:c="urn:ietf:params:xml:ns:caldav">
                <d:prop>
                    <d:getetag />
                    <c:calendar-data />
                </d:prop>
                <c:filter>
                    <c:comp-filter name="VCALENDAR">
                        <c:comp-filter name="VEVENT">
                            <c:time-range start="{start_str}" end="{end_str}" />
                        </c:comp-filter>
                    </c:comp-filter>
                </c:filter>
            </c:calendar-query>"""
            
        res = requests.request("REPORT", url, auth=self.auth, headers=headers, data=body, timeout=10)
        
        if res.status_code not in (200, 207):
            return [f"Failed to fetch events: HTTP {res.status_code}"]
            
        events = []
        if "BEGIN:VEVENT" in res.text:
            raw_events = res.text.split("BEGIN:VEVENT")[1:]
            for ev in raw_events:
                events.append("BEGIN:VEVENT" + ev[:ev.find("END:VEVENT")+10])
                
        return events if events else ["No upcoming events found."]

    def create_event(self, title: str, start_iso: str, end_iso: str) -> bool:
        """Create a new calendar event (.ics)."""
        uid = str(uuid.uuid4())
        
        # Strip dashes and colons for standard ICS basic format
        start_fmt = start_iso.replace("-", "").replace(":", "").replace(".000", "")
        end_fmt = end_iso.replace("-", "").replace(":", "").replace(".000", "")
        
        if "Z" not in start_fmt: start_fmt += "Z"
        if "Z" not in end_fmt: end_fmt += "Z"
        
        ics = f"BEGIN:VCALENDAR\r\nVERSION:2.0\r\nBEGIN:VEVENT\r\nUID:{uid}\r\nSUMMARY:{title}\r\nDTSTART:{start_fmt}\r\nDTEND:{end_fmt}\r\nEND:VEVENT\r\nEND:VCALENDAR\r\n"
        
        url = f"{self._discover_calendar_url()}/{uid}.ics"
        headers = {"Content-Type": "text/calendar; charset=utf-8", "If-None-Match": "*"}
        
        res = requests.put(url, auth=self.auth, headers=headers, data=ics.encode('utf-8'), timeout=10)
        
        if res.status_code not in (200, 201, 204):
            raise RuntimeError(f"Failed to create event: HTTP {res.status_code} - {res.text}")
        return True
