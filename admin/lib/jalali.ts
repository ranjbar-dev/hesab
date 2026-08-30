import { toGregorian, toJalaali } from "jalaali-js";
export const FA_MONTHS=["فروردین","اردیبهشت","خرداد","تیر","مرداد","شهریور","مهر","آبان","آذر","دی","بهمن","اسفند"];
const fa=(n:number)=>n.toLocaleString("fa-IR",{useGrouping:false}).padStart(2,"۰");
export function isoToJalaliLabel(iso:string){const d=new Date(iso),j=toJalaali(d.getUTCFullYear(),d.getUTCMonth()+1,d.getUTCDate());return `${j.jy.toLocaleString("fa-IR",{useGrouping:false})}/${fa(j.jm)}/${fa(j.jd)}`}
export function jalaliDayRange(jy:number,jm:number,jd:number){const g=toGregorian(jy,jm,jd),from=new Date(Date.UTC(g.gy,g.gm-1,g.gd)),to=new Date(from.getTime()+86400000);return{from:from.toISOString(),to:to.toISOString()}}
