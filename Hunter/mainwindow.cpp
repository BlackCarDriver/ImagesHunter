#include "mainwindow.h"
#include "ui_mainwindow.h"
#include <QMenu>


MainWindow::MainWindow(QWidget *parent) : QMainWindow(parent) , ui(new Ui::MainWindow){
    setWindowIcon(QIcon(":./icom.ico"));
    ui->setupUi(this);

    //menu bar setting
    QMenu* file = menuBar()->addMenu(tr("文件(&F)"));
    QAction *openconf = new QAction(tr("打开配置(&O)"), this);
    QAction *saveconf = new QAction(tr("保存配置(&S)"), this);
    QAction *quit = new QAction(tr("退出(&Q)"), this);
    file->addAction(openconf);
    file->addAction(saveconf);
    file->addAction(quit);
    QMenu* help = menuBar()->addMenu(tr("帮助(&H)"));
    QAction *connect = new QAction(tr("联系作者(&C)"), this);
    QAction *seehelp = new QAction(tr("查看帮助(&S)"), this);
    QAction *seeversion = new QAction(tr("版本信息(&V)"), this);
    help->addAction(seehelp);
    help->addAction(connect);
    help->addAction(seeversion);
    QMenu* lanugh = menuBar()->addMenu(tr("语言(&L)"));
    QAction *chinese = new QAction(tr("中文(&C)"), this);
    QAction *english = new QAction(tr("&English"), this);
    lanugh-> addAction(english);
    lanugh->addAction(chinese);
}


MainWindow::~MainWindow(){
    delete ui;
}

