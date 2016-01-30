CREATE TABLE `submissionContents` (
  `hash` binary(32) NOT NULL,
  `content` mediumblob NOT NULL,
  PRIMARY KEY (`hash`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

CREATE TABLE `logEntries` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `submissionHash` binary(32) NOT NULL,
  `rootDN` varchar(255) NOT NULL,
  `entryNum` bigint(20) NOT NULL,
  `logID` binary(32) NOT NULL,
  `entryType` tinyint(8) NOT NULL,
  `unparseableComponent` tinyint(1) NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `entryNum_logID` (`entryNum`, `logID`),
  UNIQUE KEY `submissionHash_logID` (`submissionHash`, `logID`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
