#yeav -- yet another evio

这个项目是根据 [evio](https://github.com/tidwall/evio) 进行仿写的，对 [evio](https://github.com/tidwall/evio) 的API进行了少量的更改。
详见 [examples](https://github.com/qaqslzy/yaev/tree/master/examples) 。

此项目对evio的惊群均衡负载进行了修改，通过使用一个实现listenLoop，注册listener，从而实现均衡负载。避免了原项目中的惊群现象。