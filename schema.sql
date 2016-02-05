CREATE TABLE `certificates` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `hash` binary(32) UNIQUE NOT NULL,
  `der` mediumblob NOT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

CREATE TABLE `logEntries` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `chainIDs` varchar(255) NOT NULL,
  `rootDN` varchar(255) NOT NULL,
  `entryNum` bigint(20) NOT NULL,
  `logID` binary(32) NOT NULL,
  `entryType` tinyint(8) NOT NULL,
  `unparseableComponent` tinyint(1) NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `entryNum_logID` (`entryNum`, `logID`),
  UNIQUE KEY `chainIDs_logID` (`chainIDs`, `logID`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
