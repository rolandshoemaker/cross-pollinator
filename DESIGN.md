
table validRoots
    dn
    log id

table logEntries
    hash
    root dn/hash
    log id
    entry number

three concurrent sections, one that retrieves new entries
from each log and parses them, another which finds entries
that could be submitted to other logs, and a final one
that submits these entries to the relevant logs.

the first and last should be concurrent for each log and
feed into the db/eat from a channel. the middle bit should
feed the last channel and, using SQL queries, find the magic
entries for submission.
