# Zoop

A daily journal app: each user writes on paper-like pages, one per notebook per day, starting from the day they signed up.

## Language

### Writing

**Notebook**:
One tab on the side divider; a separate stream of daily writing. v1 has two: *My Journal* and *I Learned*.
_Avoid_: Tab (that's the UI control, not the thing), category, section

**Page**:
One Notebook's writing for one Day.
_Avoid_: Note, entry, post

**Day**:
A calendar date in the user's timezone. Each user's first Day is the date they signed up.
_Avoid_: Date (when meaning the journaling unit)

**Written Day**:
A Day on which at least one Page has some text. A Page that has been emptied doesn't count. Days that aren't Written Days appear faded in the rail.
_Avoid_: Active day, entry day

**Grace Window**:
The first 2 hours after midnight, during which the previous Day's Pages can still be written.
_Avoid_: Late period, buffer

**Title**:
The first line of a Day's *My Journal* Page, shown bold on the Page and as plain text on that Day's Island. It is always a plain line, never a list item or quote.
_Avoid_: Heading, subject

**Island**:
The thin pill for one Day in the side date rail, showing that Day's date and title.
_Avoid_: Date chip, bar

**Notice**:
A short message about the app, the account or the open Page (for example "Not saved, retrying"), shown in the notice pill at the top of the sheet. Notices stack most urgent first.
_Avoid_: Banner, toast, alert, Island (an Island is always a Day in the rail)

### Access

**Free User / Premium User**:
Free Users can write only today's Pages (plus the previous Day's Pages during the Grace Window). Premium Users can edit past Pages and see edit history (not in v1).

**Deactivated Account**:
An account its user has closed. Its journal is hidden at once and erased within a year. It can't be signed into, and its email can't sign up again.
_Avoid_: Deleted account, closed account
