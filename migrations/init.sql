CREATE TABLE `link` (
    `id` int(11) NOT NULL AUTO_INCREMENT,
    `link` varchar(32) NOT NULL,
    `date_created` datetime NOT NULL,
    `date_expired` datetime NOT NULL,
    PRIMARY KEY (`id`),
    UNIQUE KEY `link` (`link`)
) ENGINE=InnoDB  DEFAULT CHARSET=utf8;

CREATE TABLE `redirect` (
    `id` int(11) NOT NULL AUTO_INCREMENT,
    `link_id` int(11) NOT NULL,
    `redirect_uri` varchar(1024) NOT NULL,
    PRIMARY KEY (`id`),
    CONSTRAINT `link_fk1` FOREIGN KEY (`link_id`) REFERENCES `link` (`id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
