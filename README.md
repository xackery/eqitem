# eqitem
import items from sodaeq to eqemu

To use:

* place eqemu_config.json inside the same directory as eqitem.exe, or set the EQEMU_CONFIG environment variable to where your eqemu_config is located
* download http://items.sodeq.org/downloads/items.txt.gz and extract. (~111 mb)
* run the program, such as `eqitem.exe C:\Downloads\items.txt`
* by default, it will insert any missing item id's into your database. You can optionally provide an itemid, e.g. `eqitem.exe items.txt 1234` to only insert 1234. (It will only do so if the item id does not exist)

usage: eqitem items.txt [itemid]