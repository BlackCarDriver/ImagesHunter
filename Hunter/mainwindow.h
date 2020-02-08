#ifndef MAINWINDOW_H
#define MAINWINDOW_H
#include "bridge.h"
#include <QMainWindow>
#include <vector>

QT_BEGIN_NAMESPACE
namespace Ui { class MainWindow; }
QT_END_NAMESPACE

class MainWindow : public QMainWindow{
    Q_OBJECT

public:
    MainWindow(QWidget *parent = nullptr);
    ~MainWindow();

private: //消息处理函数
    int test(QString content);
    int errorHandle(QString content);
    int infoHandle(QString content);
    int tableHandle(QString content);
    int staticHandle(QString content);

private slots:
    void on_btn_start_clicked();
    void messageHandle(QString key, QString content);
    void functionHandle(QString key);

    void on_pushButton_clicked();

    void on_btn_stop_clicked();

private:
    Ui::MainWindow *ui;
    Bridge<MainWindow>*bridge;
    vector<QWidget*> widgetArray;
    void setWidgetState(bool);
    QString getConfig();
};

#endif // MAINWINDOW_H

